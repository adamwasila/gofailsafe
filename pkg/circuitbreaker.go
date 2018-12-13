package failsafe

import (
	"fmt"
	"time"
)

type CircuitBreaker struct {
	failureThreshold     int
	successThreshold     int
	openDelay            time.Duration
	recover              bool
	failuresCount        int
	successCount         int
	state                circuitBreakerState
	stateChangeTimestamp time.Time
}

type circuitBreakerOption func(*CircuitBreaker) error

type circuitBreakerState int

const (
	stateClosed = circuitBreakerState(iota)
	stateOpen
	stateHalfOpen
)

func NewCircuitBreaker(options ...circuitBreakerOption) (*CircuitBreaker, error) {
	cb := &CircuitBreaker{
		failureThreshold:     1,
		successThreshold:     1,
		openDelay:            1 * time.Minute,
		recover:              false,
		failuresCount:        0,
		successCount:         0,
		state:                stateClosed,
		stateChangeTimestamp: time.Now(),
	}
	for _, o := range options {
		err := o(cb)
		if err != nil {
			return nil, err
		}
	}
	return cb, nil
}

func (cb *CircuitBreaker) State() string {
	if cb.state == stateOpen && time.Since(cb.stateChangeTimestamp) > cb.openDelay {
		cb.state = stateHalfOpen
		cb.stateChangeTimestamp = time.Now()
	}
	switch cb.state {
	case stateClosed:
		return "CLOSED"
	case stateHalfOpen:
		return "HALF-OPEN"
	case stateOpen:
		return "OPEN"
	default:
		return "???"
	}
}

func FailureThreshold(failureThreshold int) circuitBreakerOption {
	return func(cb *CircuitBreaker) error {
		cb.failureThreshold = failureThreshold
		if failureThreshold < 1 {
			return fmt.Errorf("Failure threshold must be postive (is: %v)", failureThreshold)
		}
		return nil
	}
}

func SuccessThreshold(successThreshold int) circuitBreakerOption {
	return func(cb *CircuitBreaker) error {
		cb.successThreshold = successThreshold
		if successThreshold < 1 {
			return fmt.Errorf("Success threshold must be postive (is: %v)", successThreshold)
		}
		return nil
	}
}

func OpenDelay(openDelay time.Duration) circuitBreakerOption {
	return func(cb *CircuitBreaker) error {
		cb.openDelay = openDelay
		if openDelay < 0 {
			return fmt.Errorf("Delay must be >= 0 (is: %v)", openDelay)
		}
		return nil
	}
}

func OpenOnPanic() circuitBreakerOption {
	return func(r *CircuitBreaker) error {
		r.recover = true
		return nil
	}
}

func (cb *CircuitBreaker) Run(job func() error) error {
	if cb.recover {
		job = recoverDecorator(job)
	}
	cb.State()
	switch cb.state {
	case stateClosed:
		err := job()
		if err != nil {
			cb.failuresCount++
		} else {
			cb.failuresCount = 0
		}
		if cb.failuresCount >= cb.failureThreshold {
			cb.failuresCount = 0
			cb.state = stateOpen
			cb.stateChangeTimestamp = time.Now()
		}
		return err
	case stateHalfOpen:
		err := job()
		if err == nil {
			cb.successCount++
			if cb.successCount >= cb.successThreshold {
				cb.successCount = 0
				cb.state = stateClosed
				cb.stateChangeTimestamp = time.Now()
			}
		} else {
			cb.successCount = 0
			cb.state = stateOpen
			cb.stateChangeTimestamp = time.Now()
		}
		return err
	case stateOpen:
		return fmt.Errorf("Error: circuit breaker is open")
	default:
		panic("Must never happen")
	}
}
