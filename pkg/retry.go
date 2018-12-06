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

type retryOption func(*Retry)

func NewRetry(options ...retryOption) *Retry {
	r := &Retry{
		delay:   1 * time.Second,
		retries: 3,
		predicate: func(result interface{}, err error) bool {
			return err != nil
		},
		recover: false,
	}
	for _, o := range options {
		o(r)
	}
	return r
}

func WithDelay(delay time.Duration) retryOption {
	return func(r *Retry) {
		r.delay = delay
	}
}

func WithRetries(retries int) retryOption {
	return func(r *Retry) {
		r.retries = retries
	}
}

func RetryIf(predicate func(result interface{}, err error) bool) retryOption {
	return func(r *Retry) {
		r.predicate = predicate
	}
}

func RetryOnPanic() retryOption {
	return func(r *Retry) {
		r.recover = true
	}
}

type Job func() error

type ResultJob func() (interface{}, error)

func recoverDecorator(job Job) func() (err error) {
	return func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("Panic: %v", r)
			}
		}()
		return job()
	}
}

func (r *Retry) Run(job Job) {
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

func (r *Retry) Get(job ResultJob) {
	i := 1
	for {
		if res, err := job(); !r.predicate(res, err) || i >= r.retries {
			break
		}
		i++
		time.Sleep(r.delay)
	}
}
