package thread

import (
	"fmt"
	"runtime"
)

func NoPanic(f func() error) func() error {
	return func() (err error) {
		defer func() {
			err = PanicToError(recover(), err)
		}()
		return f()
	}
}

func PanicToError(thrown interface{}, defaultErr error) error {
	if thrown == nil {
		return defaultErr
	}
	const size = 64 << 10
	trace := make([]byte, size)
	trace = trace[:runtime.Stack(trace, false)]
	err, ok := thrown.(error)
	if !ok {
		err = fmt.Errorf("panic: %v\n%s", thrown, string(trace))
	} else {
		err = fmt.Errorf("panic: %w\n%s", err, string(trace))
	}
	return err
}
