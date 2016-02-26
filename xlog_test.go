package xlog

import (
	"bytes"
	"log"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var fakeNow = time.Date(0, 0, 0, 0, 0, 0, 0, time.Local)
var critialLoggerMux = sync.Mutex{}

func init() {
	now = func() time.Time {
		return fakeNow
	}
}

func TestNew(t *testing.T) {
	oc := NewOutputChannel(newTestOutput())
	defer oc.Close()
	c := Config{
		Level:  LevelError,
		Output: oc,
		Fields: F{"foo": "bar"},
	}
	L := New(c)
	l, ok := L.(*logger)
	if assert.True(t, ok) {
		assert.Equal(t, LevelError, l.level)
		assert.Equal(t, c.Output, l.output)
		assert.Equal(t, F{"foo": "bar"}, F(l.fields))
		// Ensure l.fields is a clone
		c.Fields["bar"] = "baz"
		assert.Equal(t, F{"foo": "bar"}, F(l.fields))
		l.close()
	}
}

func TestCopy(t *testing.T) {
	oc := NewOutputChannel(newTestOutput())
	defer oc.Close()
	c := Config{
		Level:  LevelError,
		Output: oc,
		Fields: F{"foo": "bar"},
	}
	l := New(c).(*logger)
	l2 := Copy(l).(*logger)
	assert.Equal(t, l.output, l2.output)
	assert.Equal(t, l.level, l2.level)
	assert.Equal(t, l.fields, l2.fields)
	l2.SetField("bar", "baz")
	assert.Equal(t, map[string]interface{}{"foo": "bar"}, l.fields)
	assert.Equal(t, map[string]interface{}{"foo": "bar", "bar": "baz"}, l2.fields)

	assert.Equal(t, NopLogger, Copy(NopLogger))
	assert.Equal(t, NopLogger, Copy(nil))
}

func TestNewDefautOutput(t *testing.T) {
	L := New(Config{})
	l, ok := L.(*logger)
	if assert.True(t, ok) {
		assert.NotNil(t, l.output)
		l.close()
	}
}

func TestSend(t *testing.T) {
	o := newTestOutput()
	l := New(Config{Output: o}).(*logger)
	l.send(LevelDebug, 1, "test", F{"foo": "bar"})
	last := <-o.w
	assert.Contains(t, last["file"], "log_test.go:")
	delete(last, "file")
	assert.Equal(t, map[string]interface{}{"time": fakeNow, "level": "debug", "message": "test", "foo": "bar"}, last)

	l.SetField("bar", "baz")
	l.send(LevelInfo, 1, "test", F{"foo": "bar"})
	last = <-o.w
	assert.Contains(t, last["file"], "log_test.go:")
	delete(last, "file")
	assert.Equal(t, map[string]interface{}{"time": fakeNow, "level": "info", "message": "test", "foo": "bar", "bar": "baz"}, last)

	l = New(Config{Output: o, Level: 1}).(*logger)
	o.reset()
	l.send(0, 2, "test", F{"foo": "bar"})
	assert.True(t, o.empty())
}

func TestSendDrop(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	critialLoggerMux.Lock()
	oldCritialLogger := critialLogger
	critialLogger = log.New(buf, "", 0)
	defer func() {
		critialLogger = oldCritialLogger
		critialLoggerMux.Unlock()
	}()
	oc := NewOutputChannelBuffer(Discard, 1)
	defer oc.Close()
	l := New(Config{Output: oc}).(*logger)
	l.send(LevelDebug, 2, "test", F{"foo": "bar"})
	l.send(LevelDebug, 2, "test", F{"foo": "bar"})
	l.send(LevelDebug, 2, "test", F{"foo": "bar"})
	for i := 0; i < 10; i++ {
		runtime.Gosched()
		if "send error: buffer fullsend error: buffer full" == buf.String() {
			return
		}
	}
	t.Fail()
}

func TestWxtractFields(t *testing.T) {
	v := []interface{}{"a", 1, map[string]interface{}{"foo": "bar"}}
	f := extractFields(&v)
	assert.Equal(t, map[string]interface{}{"foo": "bar"}, f)
	assert.Equal(t, []interface{}{"a", 1}, v)

	v = []interface{}{map[string]interface{}{"foo": "bar"}, "a", 1}
	f = extractFields(&v)
	assert.Nil(t, f)
	assert.Equal(t, []interface{}{map[string]interface{}{"foo": "bar"}, "a", 1}, v)

	v = []interface{}{"a", 1, F{"foo": "bar"}}
	f = extractFields(&v)
	assert.Equal(t, map[string]interface{}{"foo": "bar"}, f)
	assert.Equal(t, []interface{}{"a", 1}, v)

	v = []interface{}{}
	f = extractFields(&v)
	assert.Nil(t, f)
	assert.Equal(t, []interface{}{}, v)
}

func TestDebug(t *testing.T) {
	o := newTestOutput()
	l := New(Config{Output: o}).(*logger)
	l.Debug("test", F{"foo": "bar"})
	last := <-o.w
	assert.Contains(t, last["file"], "log_test.go:")
	delete(last, "file")
	assert.Equal(t, map[string]interface{}{"time": fakeNow, "level": "debug", "message": "test", "foo": "bar"}, last)
}

func TestDebugf(t *testing.T) {
	o := newTestOutput()
	l := New(Config{Output: o}).(*logger)
	l.Debugf("test %d", 1, F{"foo": "bar"})
	last := <-o.w
	assert.Contains(t, last["file"], "log_test.go:")
	delete(last, "file")
	assert.Equal(t, map[string]interface{}{"time": fakeNow, "level": "debug", "message": "test 1", "foo": "bar"}, last)
}

func TestInfo(t *testing.T) {
	o := newTestOutput()
	l := New(Config{Output: o}).(*logger)
	l.Info("test", F{"foo": "bar"})
	last := <-o.w
	assert.Contains(t, last["file"], "log_test.go:")
	delete(last, "file")
	assert.Equal(t, map[string]interface{}{"time": fakeNow, "level": "info", "message": "test", "foo": "bar"}, last)
}

func TestInfof(t *testing.T) {
	o := newTestOutput()
	l := New(Config{Output: o}).(*logger)
	l.Infof("test %d", 1, F{"foo": "bar"})
	last := <-o.w
	assert.Contains(t, last["file"], "log_test.go:")
	delete(last, "file")
	assert.Equal(t, map[string]interface{}{"time": fakeNow, "level": "info", "message": "test 1", "foo": "bar"}, last)
}

func TestWarn(t *testing.T) {
	o := newTestOutput()
	l := New(Config{Output: o}).(*logger)
	l.Warn("test", F{"foo": "bar"})
	last := <-o.w
	assert.Contains(t, last["file"], "log_test.go:")
	delete(last, "file")
	assert.Equal(t, map[string]interface{}{"time": fakeNow, "level": "warn", "message": "test", "foo": "bar"}, last)
}

func TestWarnf(t *testing.T) {
	o := newTestOutput()
	l := New(Config{Output: o}).(*logger)
	l.Warnf("test %d", 1, F{"foo": "bar"})
	last := <-o.w
	assert.Contains(t, last["file"], "log_test.go:")
	delete(last, "file")
	assert.Equal(t, map[string]interface{}{"time": fakeNow, "level": "warn", "message": "test 1", "foo": "bar"}, last)
}

func TestError(t *testing.T) {
	o := newTestOutput()
	l := New(Config{Output: o}).(*logger)
	l.Error("test", F{"foo": "bar"})
	last := <-o.w
	assert.Contains(t, last["file"], "log_test.go:")
	delete(last, "file")
	assert.Equal(t, map[string]interface{}{"time": fakeNow, "level": "error", "message": "test", "foo": "bar"}, last)
}

func TestErrorf(t *testing.T) {
	o := newTestOutput()
	l := New(Config{Output: o}).(*logger)
	l.Errorf("test %d%v", 1, F{"foo": "bar"})
	last := <-o.w
	assert.Contains(t, last["file"], "log_test.go:")
	delete(last, "file")
	assert.Equal(t, map[string]interface{}{"time": fakeNow, "level": "error", "message": "test 1", "foo": "bar"}, last)
}

func TestFatal(t *testing.T) {
	e := exit1
	exited := 0
	exit1 = func() { exited++ }
	defer func() { exit1 = e }()
	o := newTestOutput()
	l := New(Config{Output: NewOutputChannel(o)}).(*logger)
	l.Fatal("test", F{"foo": "bar"})
	last := <-o.w
	assert.Contains(t, last["file"], "log_test.go:")
	delete(last, "file")
	assert.Equal(t, map[string]interface{}{"time": fakeNow, "level": "fatal", "message": "test", "foo": "bar"}, last)
	assert.Equal(t, 1, exited)
}

func TestFatalf(t *testing.T) {
	e := exit1
	exited := 0
	exit1 = func() { exited++ }
	defer func() { exit1 = e }()
	o := newTestOutput()
	l := New(Config{Output: NewOutputChannel(o)}).(*logger)
	l.Fatalf("test %d%v", 1, F{"foo": "bar"})
	last := <-o.w
	assert.Contains(t, last["file"], "log_test.go:")
	delete(last, "file")
	assert.Equal(t, map[string]interface{}{"time": fakeNow, "level": "fatal", "message": "test 1", "foo": "bar"}, last)
	assert.Equal(t, 1, exited)
}

func TestWrite(t *testing.T) {
	o := newTestOutput()
	xl := New(Config{Output: NewOutputChannel(o)}).(*logger)
	l := log.New(xl, "prefix ", 0)
	l.Printf("test")
	last := <-o.w
	assert.Contains(t, last["file"], "log_test.go:")
	delete(last, "file")
	assert.Equal(t, map[string]interface{}{"time": fakeNow, "level": "info", "message": "prefix test"}, last)
}
