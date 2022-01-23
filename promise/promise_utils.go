package promise

import (
	"errors"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
)

// All is used to return all results when all promises have resolved. If one promise rejects, the error will be returned.
func All[T any](promises ...*Promise[T]) ([]T, error) {
	// Defines the wait group.
	wg := errgroup.Group{}

	// Defines the results.
	results := make([]T, len(promises))

	// Go through each promise and hook handlers to it.
	for i, p := range promises {
		ptr := &results[i]
		x := p
		wg.Go(func() error {
			errChan := make(chan error)
			Then(x, func(res T) (struct{}, error) {
				*ptr = res
				errChan <- nil
				return struct{}{}, nil
			})
			Catch(x, func(err error) (struct{}, error) {
				errChan <- err
				return struct{}{}, nil
			})
			return <-errChan
		})
	}

	// Wait for the channel.
	return results, wg.Wait()
}

// NoPromises is used for Race where it is expected that promises will be set.
var NoPromises = errors.New("no promises specified")

// Race returns the result of the first promise to resolve.
func Race[T any](promises ...*Promise[T]) (T, error) {
	// If there's no promises, return here.
	if len(promises) == 0 {
		var x T
		return x, NoPromises
	}

	// Wait for the first promise to resolve.
	var done uintptr
	errorCh := make(chan error)
	var res T
	for _, p := range promises {
		Then(p, func(innerRes T) (struct{}, error) {
			if atomic.SwapUintptr(&done, 1) == 1 {
				return struct{}{}, nil
			}
			res = innerRes
			errorCh <- nil
			return struct{}{}, nil
		})
		Catch(p, func(innerErr error) (struct{}, error) {
			if atomic.SwapUintptr(&done, 1) == 1 {
				return struct{}{}, nil
			}
			errorCh <- innerErr
			return struct{}{}, nil
		})
	}
	return res, <-errorCh
}

// Iterator is used to create a function to iterate over promises. Next will block until the next promise resolves.
// Note the next function is not thread safe!
func Iterator[T any](promises ...*Promise[T]) func() (val T, end bool, err error) {
	index := 0
	return func() (val T, end bool, err error) {
		if index == len(promises) {
			// We have exhausted all promises.
			end = true
			return
		}

		// Get the next promise.
		p := promises[index]
		index++

		// Try the fast path.
		if res := p.Resolve(); res != nil {
			return res.Result, false, res.Error
		}

		// Go the hook path.
		waitCh := make(chan struct{})
		Then(p, func(res T) (struct{}, error) {
			val = res
			waitCh <- struct{}{}
			return struct{}{}, nil
		})
		Catch(p, func(innerErr error) (struct{}, error) {
			err = innerErr
			waitCh <- struct{}{}
			return struct{}{}, nil
		})
		<-waitCh
		return
	}
}
