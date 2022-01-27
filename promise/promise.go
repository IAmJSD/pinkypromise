package promise

import (
	"sync"
)

type element struct {
	value interface{}
	next  *element
}

type stack struct {
	start *element
	end   *element
}

func (l *stack) push(x interface{}) *element {
	e := &element{value: x}
	if l.start == nil {
		l.start = e
	} else {
		l.end.next = e
	}
	l.end = e
	return e
}

func (l *stack) format() {
	l.start = nil
	l.end = nil
}

// Promise is a promise that can be resolved or rejected.
// Note that manually creating this will result in blank values.
// You probably want to use .NewRejected, .NewResolved, or .NewFn instead.
type Promise[T any] struct {
	// defines the lock for the results
	lock sync.Mutex

	// defines if the promise is not done
	// the reason this is the opposite is because it will be empty on manual init
	notDone bool

	// ensures that we do not cause undefined behaviour by making things run in parallel when done
	doneMu sync.Mutex

	// Defines the result of the promise.
	res T
	err error

	// defines the then list.
	thenStack stack

	// defines the error list.
	errorStack stack
}

// Call the function and handle the results.
func (p *Promise[T]) call(f func() (T, error)) {
	// Call the function.
	res, err := f()

	// Ensures that we do not cause undefined behaviour by making things run in parallel when done
	p.lock.Lock()
	p.notDone = false
	p.err = err
	p.res = res
	thenStack := p.thenStack
	errorStack := p.errorStack
	p.thenStack.format()
	p.errorStack.format()
	p.lock.Unlock()

	// Lock and run handlers.
	p.doneMu.Lock()
	defer p.doneMu.Unlock()
	if err != nil {
		for s := errorStack.start; s != nil; s = s.next {
			s.value.(func(error))(err)
		}
		return
	}
	for s := thenStack.start; s != nil; s = s.next {
		s.value.(func(T))(res)
	}
}

// PromiseResolution is used to define the resolution of a promise.
type PromiseResolution[T any] struct {
	// Result defines the result of the promise.
	Result T

	// Error defines if the promise rejected.
	Error error
}

// Resolve is used to get the promise resolution. Returns a nil pointer if the promise is unresolved.
func (p *Promise[T]) Resolve() *PromiseResolution[T] {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.notDone {
		return nil
	}
	return &PromiseResolution[T]{Result: p.res, Error: p.err}
}

// NewFn is used to create a new function promise.
func NewFn[T any](f func() (T, error)) *Promise[T] {
	p := &Promise[T]{notDone: true}
	go p.call(f)
	return p
}

// NewFnWithArg behaves the same as NewFn but allows you to pass in an argument.
// This is useful for functions that take in an argument or where you want to pass in a context.
func NewFnWithArg[T any, X any](arg T, f func(T) (X, error)) *Promise[X] {
	return NewFn(func() (X, error) { return f(arg) })
}

// NewResolved is used to create a new resolved promise.
func NewResolved[T any](result T) *Promise[T] {
	return &Promise[T]{res: result}
}

// NewRejected is used to create a new rejected promise.
func NewRejected[T any](err error) *Promise[T] {
	return &Promise[T]{err: err}
}

// Then is used to add a then handler to the promise.
// In the event that the promise has already resolved, this will result in a new go-routine being spawned.
func Then[T any, X any](p *Promise[T], f func(T) (X, error)) *Promise[X] {
	// Lock and get all values.
	p.lock.Lock()
	done := !p.notDone
	res := p.res
	err := p.err

	// If we are not done, we should add to the handlers.
	if !done {
		// Add the then handler.
		newPromise := &Promise[X]{notDone: true}
		thenHn := func(res T) {
			newPromise.call(func() (X, error) {
				return f(res)
			})
		}
		p.thenStack.push(thenHn)

		// Add the catch handler.
		catchHn := func(err error) {
			newPromise.call(func() (_ X, innerErr error) {
				innerErr = err
				return
			})
		}
		p.errorStack.push(catchHn)

		// Now unlock the promise.
		p.lock.Unlock()

		// Return the new promise.
		return newPromise
	}

	// Unlock the root data.
	p.lock.Unlock()

	// Create a new promise function to handle this.
	return NewFn(func() (innerRes X, innerErr error) {
		// Lock the single-thread mutex to prevent undefined behaviour.
		p.doneMu.Lock()

		// Defer unlocking until this is done.
		defer p.doneMu.Unlock()

		// If there was an error, return it now.
		if err != nil {
			innerErr = err
			return
		}

		// Call the function.
		return f(res)
	})
}

// Catch is used to add a error catching handler to the promise.
// In the event that the promise has already resolved, this will result in a new go-routine being spawned.
func Catch[T any, X any](p *Promise[T], f func(error) (X, error)) *Promise[X] {
	// Lock and get all values.
	p.lock.Lock()
	done := !p.notDone
	err := p.err

	// Defines the new promise.
	newPromise := &Promise[X]{notDone: true}

	// If we are not done, we should add to the handlers.
	if !done {
		// Add the catch handler.
		catchHn := func(err error) {
			newPromise.call(func() (X, error) {
				return f(err)
			})
		}
		p.errorStack.push(catchHn)

		// Now unlock the origin promise.
		p.lock.Unlock()

		// Return the new promise.
		return newPromise
	}

	// Unlock the root data.
	p.lock.Unlock()

	// If the error was nil, mark the promise as done and return it.
	if err == nil {
		newPromise.notDone = false
		return newPromise
	}

	// Create a go-routine to handle calling the promise.
	go newPromise.call(func() (X, error) {
		// Lock the single-thread mutex to prevent undefined behaviour.
		p.doneMu.Lock()

		// Defer unlocking until this is done.
		defer p.doneMu.Unlock()

		// Call the function.
		return f(err)
	})

	// Return the promise.
	return newPromise
}
