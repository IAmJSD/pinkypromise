package promise

import (
	"errors"
	"testing"
	"time"
)

func TestAll(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		a, err := All[string]()
		if err != nil {
			t.Error("error isn't nil")
		}
		if len(a) != 0 {
			t.Error("length is wrong")
		}
	})

	t.Run("rejected", func(t *testing.T) {
		a := make([]*Promise[string], 10)
		for i := 0; i < 10; i++ {
			if i == 2 {
				a[i] = NewRejected[string](errors.New("hello world"))
			} else {
				x := i
				a[i] = NewFn(func() (string, error) {
					time.Sleep(time.Millisecond * time.Duration(x+1))
					return "hello world", nil
				})
			}
		}
		_, err := All(a...)
		if err == nil {
			t.Fatal("error is nil")
		}
		if err.Error() != "hello world" {
			t.Error("value is wrong")
		}
	})
}

func TestRace(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		_, err := Race[string]()
		if err != NoPromises {
			t.Error("no promises error not thrown")
		}
	})

	t.Run("resolve", func(t *testing.T) {
		x, err := Race(
			// Make the first promise the mid range one.
			NewFn(func() (string, error) {
				time.Sleep(time.Millisecond*2)
				return "hello world mid", nil
			}),

			// Make the mid promise the fastest one.
			NewResolved("hello world fastest"),

			// Make the end promise the slowest one.
			NewFn(func() (string, error) {
				time.Sleep(time.Millisecond*5)
				return "", errors.New("hello world mid")
			}),
		)
		if err != nil {
			t.Error("error isn't nil")
		}
		if x != "hello world fastest" {
			t.Error("value is wrong")
		}
	})

	t.Run("reject", func(t *testing.T) {
		_, err := Race(
			// Make the first promise the mid range one.
			NewFn(func() (string, error) {
				time.Sleep(time.Millisecond*2)
				return "hello world mid", nil
			}),

			// Make the mid promise the fastest one.
			NewRejected[string](errors.New("hello world fastest")),

			// Make the end promise the slowest one.
			NewFn(func() (string, error) {
				time.Sleep(time.Millisecond*5)
				return "hello world mid", nil
			}),
		)
		if err == nil {
			t.Fatal("error is nil")
		}
		if err.Error() != "hello world fastest" {
			t.Error("value is wrong")
		}
	})
}

func TestIterator(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		iteratorFn := Iterator[string]()
		_, end, err := iteratorFn()
		if err != nil {
			t.Error("error isn't nil")
		}
		if !end {
			t.Error("end is false")
		}
	})

	// Defines all promises in our iteration test.
	promises := []struct {
		name string

		promise *Promise[string]

		wantsErr string
		wantsVal string
	}{
		{
			name:     "resolved",
			promise:  NewResolved("hello world"),
			wantsVal: "hello world",
		},
		{
			name:     "rejected",
			promise:  NewRejected[string](errors.New("hello world")),
			wantsErr: "hello world",
		},
		{
			name: "pending resolve",
			promise: NewFn(func() (string, error) {
				time.Sleep(time.Millisecond * 2)
				return "hello world", nil
			}),
			wantsVal: "hello world",
		},
		{
			name: "pending reject",
			promise: NewFn(func() (string, error) {
				time.Sleep(time.Millisecond * 5)
				return "", errors.New("hello world")
			}),
			wantsErr: "hello world",
		},
	}

	// Run all the tests.
	p := make([]*Promise[string], len(promises))
	for i, v := range promises {
		p[i] = v.promise
	}
	iterator := Iterator(p...)
	for _, tt := range promises {
		t.Run(tt.name, func(t *testing.T) {
			// Call the iterator function.
			v, end, err := iterator()

			// Check if this is the end.
			if end {
				t.Error("end is in wrong place")
			}

			// Check if the error is correct.
			if tt.wantsErr == "" {
				if err != nil {
					t.Error("error isn't nil")
				}
			} else {
				if err == nil {
					t.Error("error is nil")
				} else if err.Error() != tt.wantsErr {
					t.Error("error is wrong")
				}
			}

			// Check if the value is correct.
			if v != tt.wantsVal {
				t.Error("value is wrong")
			}
		})
	}

	// Make sure this is the end.
	if _, end, _ := iterator(); !end {
		t.Error("end is in wrong place")
	}
}
