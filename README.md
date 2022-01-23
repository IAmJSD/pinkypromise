# pinkypromise
A library for creating and managing promises in the Go programming language using generics. The advantages of this library are:
- **Very high test coverage:** The `promise` package in this library has 100% test coverage, and other packages within this library are expected to have very high coverage too.
- **Compatibility promise:** The core behaviour of a promise **WILL NOT** change between all v1.x.x releases.
- **Using up to date mechanisms:** We aim to support all new language mechanisms in the future that may be useful to this library. However, we will always support the last 2 major versions of Go (Go 1.17 is an exception here due to the lack of generics support).
- **Thread safe:** This library is completely thread safe by design.
- **Predictable behaviour:** The behaviour of a promise is well documented below.

## Why would I want to use a promise?
There's no getting around it, for a lot of Go, you probably just want to call a function and wait. A promise is fairly heavy, and if you do not need the results in parallel it makes sense to either just block or use a goroutine.

However, there are some instances where you will want the results of a bunch of things at once (like for example network requests) that block for a long time. This is where having promises comes in useful. It relieves you of needing to manage the synchronisation of this, which can be annoying for racing promises or end up in a lot of duplication for waiting for all promises to resolve.

## How does the promise work?
Firstly, to use the promise you need to create the base one (all subsequent hooks will make their own promises as documented below), there are 4 main ways to make your base promise:

1. **Create a promise function:** You can use `NewFn` to create a promise based on a function. The passed through function should take no parameters and return `(T, error)` (where `T` is the type that you wish to base the promise on).
2. **Create a new resolved promise:** You can use this to pass through a promise to something that will automatically resolve to a successful result. To do this, you can use `NewResolved(<successful result>)`.
3. **Create a new rejected promise:** You can use this to pass through a promise to something that will automatically resolve to a rejection. To do this, you can use `NewRejected[T](<error>)`.
4. **Just initialize the struct:** This is mostly pretty useless unless you want a promise that's just resolves successfully for a zero value, but you can just do `&Promise[T]{}` to make a new promise.

So we have our promise, we can now do the following with it:
- **Call `Resolve` on the promise:** This function will get the current state of the promise as a struct pointer. The pointer will be nil if the promise has not resolved yet, and contain the data if it has.
- **Call `Catch` on the promise:** This function takes the promise and a function that takes in an error with a new return type allowing for the handler to return its own custom data. This will then be called if there is an error, and if not, will be ignored.
- **Call `Then` on the promise:** This function takes the promise and a function that takes in the type specified on the parent promise with a new return type allowing for the handler to return its own custom data. This will then be called if it is successful, and if not, the error will be passed to the catch handlers of this newly created promise.
- **Use a helper function to handle promises as a batch:** See below.

## How do I handle bulk promises?
So you have a bunch of promises. Great! But how do you manage them all? There are several functions to handle this:
- `All[T any](promises ...*Promise[T]) ([]T, error)`: If all promises are successful, this function waits for all promises to be done and then returns the slice of all resolved items. However, if one promise errors, the first error will immediately be returned.
- `Race[T any](promises ...*Promise[T]) (T, error)`: This function returns the first promise that was able to be resolved, whether it is successful or rejects.
- `Iterator[T any](promises ...*Promise[T]) func() (val T, end bool, err error)`: This function creates a iterator function that will block until the next promise in the arguments is done. This allows you to wait for promises as you need them. This is used like the following:
```go
promises := []*promise.Promise[string]{
    NewResolved("hello"),
    NewResolved("world"),
}
iterator := promise.Iterator()
for s, end, err := iterator(); !end; s, end, err = iterator() {
    if err != nil {
        // There was an error here that we should handle.
    }
    fmt.Println(s)
}
```
