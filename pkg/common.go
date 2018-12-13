package failsafe

import "fmt"

func recoverDecorator(job func() error) func() (err error) {
	return func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("Panic: %v", r)
			}
		}()
		return job()
	}
}
