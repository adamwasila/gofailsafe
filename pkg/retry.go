package failsafe

import (
	"fmt"
	"time"
)

type Retry struct {
	delay     time.Duration
	retries   int
	predicate func(result interface{}, err error) bool
	recover   bool
}

type retryOption func(*Retry) error

func NewRetry(options ...retryOption) (*Retry, error) {
	r := &Retry{
		delay:   0 * time.Second,
		retries: -1,
		predicate: func(result interface{}, err error) bool {
			return err != nil
		},
		recover: false,
	}
	for _, o := range options {
		err := o(r)
		if err != nil {
			return nil, err
		}
	}
	return r, nil
}

func Delay(delay time.Duration) retryOption {
	return func(r *Retry) error {
		r.delay = delay
		if delay < 0 {
			return fmt.Errorf("Delay should be >= 0 (is: %v)", delay)
		}
		return nil
	}
}

func Retries(retries int) retryOption {
	return func(r *Retry) error {
		r.retries = retries
		if retries < 0 {
			return fmt.Errorf("Number of retries should be >=0 (is: %v)", retries)
		}
		return nil
	}
}

func RetryIf(predicate func(result interface{}, err error) bool) retryOption {
	return func(r *Retry) error {
		r.predicate = predicate
		return nil
	}
}

func RetryOnPanic() retryOption {
	return func(r *Retry) error {
		r.recover = true
		return nil
	}
}

func (r *Retry) Run(job func() error) {
	if r.recover {
		job = recoverDecorator(job)
	}
	i := 1
	for {
		if err := job(); !r.predicate(nil, err) || i >= r.retries {
			break
		}
		i++
		time.Sleep(r.delay)
	}
}

	if r.recover {
		job = recoverWithResultDecorator(job)
	}
	i := 1
	for {
		if res, err := job(); !r.predicate(res, err) || i >= r.retries {
			break
		}
		i++
		time.Sleep(r.delay)
	}
}
