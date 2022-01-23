package promise

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPromise_Resolve(t *testing.T) {
	t.Run("unresolved", func(t *testing.T) {
		p := &Promise[string]{notDone: true}
		if p.Resolve() != nil {
			t.Error("promise should be unresolved")
		}
		if !p.lock.TryLock() {
			t.Error("mutex not unlocked")
		}
	})

	t.Run("error", func(t *testing.T) {
		p := &Promise[string]{err: errors.New("hello world")}
		res := p.Resolve()
		if res == nil {
			t.Error("promise should be resolved")
		}
		if res.Error == nil {
			t.Error("error is nil")
		}
		if res.Error.Error() != "hello world" {
			t.Error("error is wrong")
		}
		if !p.lock.TryLock() {
			t.Error("mutex not unlocked")
		}
	})

	t.Run("resolved", func(t *testing.T) {
		p := &Promise[string]{res: "hello world"}
		res := p.Resolve()
		if res == nil {
			t.Error("promise should be resolved")
		}
		if res.Error != nil {
			t.Error("error is not nil")
		}
		if res.Result != "hello world" {
			t.Error("result is wrong")
		}
		if !p.lock.TryLock() {
			t.Error("mutex not unlocked")
		}
	})
}

func TestNewFn(t *testing.T) {
	t.Run("resolved", func(t *testing.T) {
		p := NewFn(func() (string, error) {
			time.Sleep(time.Millisecond * 10)
			return "hello world", nil
		})
		res := p.Resolve()
		if res != nil {
			t.Error("promise should be un-resolved")
		}
		time.Sleep(time.Millisecond * 15)
		res = p.Resolve()
		if res == nil {
			t.Error("promise should be resolved")
		}
		if res.Error != nil {
			t.Error("error is not nil")
		}
		if res.Result != "hello world" {
			t.Error("result is wrong")
		}
	})

	t.Run("rejected", func(t *testing.T) {
		p := NewFn(func() (string, error) {
			time.Sleep(time.Millisecond * 10)
			return "", errors.New("hello world")
		})
		res := p.Resolve()
		if res != nil {
			t.Error("promise should be un-resolved")
		}
		time.Sleep(time.Millisecond * 15)
		res = p.Resolve()
		if res == nil {
			t.Error("promise should be resolved")
		}
		if res.Error == nil {
			t.Error("error is nil")
		}
		if res.Error.Error() != "hello world" {
			t.Error("error is wrong")
		}
		if res.Result != "" {
			t.Error("result is wrong")
		}
	})

	t.Run("then chain", func(t *testing.T) {
		// Create the then handler.
		p := NewFn(func() (string, error) {
			time.Sleep(time.Millisecond * 10)
			return "hello world", nil
		})
		var (
			a     = []int{}
			aLock = sync.Mutex{}
		)
		p.lock.Lock()
		p.thenList.PushBack(func(s string) {
			if s != "hello world" {
				t.Error("result is wrong")
			}
			aLock.Lock()
			a = append(a, 1)
			aLock.Unlock()
		})
		p.thenList.PushBack(func(s string) {
			if s != "hello world" {
				t.Error("result is wrong")
			}
			aLock.Lock()
			a = append(a, 2)
			aLock.Unlock()
		})
		p.thenList.PushBack(func(s string) {
			if s != "hello world" {
				t.Error("result is wrong")
			}
			aLock.Lock()
			a = append(a, 3)
			aLock.Unlock()
		})
		p.lock.Unlock()

		// Resolve the promise.
		time.Sleep(time.Millisecond * 15)
		res := p.Resolve()
		if res == nil {
			t.Error("promise should be resolved")
		}
		if res.Error != nil {
			t.Error("error is not nil")
		}
		if res.Result != "hello world" {
			t.Error("result is wrong")
		}

		// Check that the promises were invoked in the right order.
		aLock.Lock()
		if a[0] != 1 || a[1] != 2 || a[2] != 3 {
			t.Error("then handlers not invoked in right order")
		}
	})

	t.Run("catch chain", func(t *testing.T) {
		// Create the catch handler.
		p := NewFn(func() (string, error) {
			time.Sleep(time.Millisecond * 10)
			return "", errors.New("hello world")
		})
		var (
			a     = []int{}
			aLock = sync.Mutex{}
		)
		p.lock.Lock()
		p.errorList.PushBack(func(e error) {
			if e.Error() != "hello world" {
				t.Error("result is wrong")
			}
			aLock.Lock()
			a = append(a, 1)
			aLock.Unlock()
		})
		p.errorList.PushBack(func(e error) {
			if e.Error() != "hello world" {
				t.Error("result is wrong")
			}
			aLock.Lock()
			a = append(a, 2)
			aLock.Unlock()
		})
		p.errorList.PushBack(func(e error) {
			if e.Error() != "hello world" {
				t.Error("result is wrong")
			}
			aLock.Lock()
			a = append(a, 3)
			aLock.Unlock()
		})
		p.lock.Unlock()

		// Resolve the promise.
		time.Sleep(time.Millisecond * 15)
		res := p.Resolve()
		if res == nil {
			t.Error("promise should be resolved")
		}
		if res.Error == nil {
			t.Error("error is nil")
		}
		if res.Error.Error() != "hello world" {
			t.Error("error is wrong")
		}
		if res.Result != "" {
			t.Error("result is wrong")
		}

		// Check that the promises were invoked in the right order.
		aLock.Lock()
		if a[0] != 1 || a[1] != 2 || a[2] != 3 {
			t.Error("then handlers not invoked in right order")
		}
	})
}

func TestNewResolved(t *testing.T) {
	p := NewResolved("hello world!")
	if p.notDone {
		t.Error("promise is marked as done")
	}
	if p.err != nil {
		t.Error("error is not nil")
	}
	if p.res != "hello world!" {
		t.Error("string not correct")
	}
}

func TestNewRejected(t *testing.T) {
	p := NewRejected[string](errors.New("hello world"))
	if p.notDone {
		t.Error("promise is marked as done")
	}
	if p.err == nil {
		t.Error("error is nil")
	}
	if p.err.Error() != "hello world" {
		t.Error("string not correct")
	}
}

func TestThen(t *testing.T) {
	t.Run("error passthrough", func(t *testing.T) {
		p := NewFn(func() (string, error) {
			time.Sleep(time.Millisecond * 10)
			return "", errors.New("hello world")
		})
		x := Then(p, func(s string) (int, error) {
			return 0, nil
		})
		time.Sleep(time.Millisecond * 15)
		res := x.Resolve()
		if res == nil {
			t.Fatal("promise should be resolved")
		}
		if res.Error == nil {
			t.Fatal("error is nil")
		}
		if res.Error.Error() != "hello world" {
			t.Error("error is wrong")
		}
	})

	t.Run("promise pending", func(t *testing.T) {
		p := NewFn(func() (string, error) {
			time.Sleep(time.Millisecond * 10)
			return "hello world", nil
		})
		var (
			x     = []int{}
			xLock = sync.Mutex{}
		)
		Then(p, func(s string) (int, error) {
			if s != "hello world" {
				t.Error("result is wrong")
			}
			xLock.Lock()
			x = append(x, 1)
			xLock.Unlock()
			return 0, nil
		})
		var val uintptr
		y := Then(p, func(s string) (int, error) {
			if s != "hello world" {
				t.Error("result is wrong")
			}
			xLock.Lock()
			x = append(x, 2)
			xLock.Unlock()
			return 10, nil
		})
		Then(y, func(i int) (struct{}, error) {
			atomic.StoreUintptr(&val, uintptr(i))
			return struct{}{}, nil
		})
		time.Sleep(time.Millisecond * 15)
		xLock.Lock()
		if x[0] != 1 || x[1] != 2 {
			t.Error("promise order wrong")
		}
		if atomic.LoadUintptr(&val) != 10 {
			t.Error("tailed promise not ran")
		}
	})

	t.Run("promise resolved", func(t *testing.T) {
		x := NewResolved("hello world")
		y := Then(x, func(s string) (int, error) {
			if s != "hello world" {
				t.Error("value is wrong")
			}
			return 10, nil
		})
		time.Sleep(time.Millisecond)
		res := y.Resolve()
		if res == nil {
			t.Fatal("promise is unresolved")
		}
		if res.Error != nil {
			t.Error("error is not nil")
		}
		if res.Result != 10 {
			t.Error("invalid result")
		}
	})

	t.Run("promise rejected", func(t *testing.T) {
		x := NewRejected[string](errors.New("hello world"))
		y := Then(x, func(s string) (int, error) {
			if s != "hello world" {
				t.Error("value is wrong")
			}
			return 10, nil
		})
		time.Sleep(time.Millisecond)
		res := y.Resolve()
		if res == nil {
			t.Fatal("promise is unresolved")
		}
		if res.Error == nil {
			t.Fatal("error is nil")
		}
		if res.Error.Error() != "hello world" {
			t.Error("error wrong")
		}
	})
}

func TestCatch(t *testing.T) {
	t.Run("promise resolved", func(t *testing.T) {
		p := NewFn(func() (string, error) {
			time.Sleep(time.Millisecond * 10)
			return "hello world", nil
		})
		var called uintptr
		Catch(p, func(e error) (int, error) {
			atomic.StoreUintptr(&called, 1)
			return 0, nil
		})
		time.Sleep(time.Millisecond * 15)
		if atomic.LoadUintptr(&called) != 0 {
			t.Error("function was called")
		}
	})

	t.Run("promise rejected", func(t *testing.T) {
		p := NewFn(func() (string, error) {
			time.Sleep(time.Millisecond * 10)
			return "", errors.New("hello world")
		})
		var (
			x     = []int{}
			xLock = sync.Mutex{}
		)
		Catch(p, func(e error) (int, error) {
			if e.Error() != "hello world" {
				t.Error("error is wrong")
			}
			xLock.Lock()
			x = append(x, 1)
			xLock.Unlock()
			return 0, nil
		})
		var val uintptr
		y := Catch(p, func(e error) (int, error) {
			if e.Error() != "hello world" {
				t.Error("error is wrong")
			}
			xLock.Lock()
			x = append(x, 2)
			xLock.Unlock()
			return 10, nil
		})
		Then(y, func(i int) (struct{}, error) {
			atomic.StoreUintptr(&val, 1)
			return struct{}{}, nil
		})
		time.Sleep(time.Millisecond * 15)
		xLock.Lock()
		if x[0] != 1 || x[1] != 2 {
			t.Error("promise order wrong")
		}
		if atomic.LoadUintptr(&val) != 1 {
			t.Error("tailed promise not ran")
		}
	})

	t.Run("done resolved promise", func(t *testing.T) {
		x := NewResolved("hello world")
		var called uintptr
		Catch(x, func(e error) (int, error) {
			atomic.StoreUintptr(&called, 1)
			return 0, nil
		})
		time.Sleep(time.Millisecond * 15)
		if atomic.LoadUintptr(&called) != 0 {
			t.Error("function was called")
		}
	})

	t.Run("done rejected promise", func(t *testing.T) {
		x := NewRejected[string](errors.New("hello world"))
		var called uintptr
		Catch(x, func(e error) (int, error) {
			atomic.StoreUintptr(&called, 1)
			return 0, nil
		})
		time.Sleep(time.Millisecond * 15)
		if atomic.LoadUintptr(&called) == 0 {
			t.Error("function was not called")
		}
	})
}
