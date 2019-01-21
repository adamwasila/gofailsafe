package failsafe

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type circuitBreakerConfig struct {
	failureThreshold    int32
	successThreshold    int32
	openDelay           time.Duration
	recover             bool
	halfOpenMaxInflight int64
}

type CircuitBreaker struct {
	*circuitBreakerConfig
	failureCount int32
	successCount int32
	inflight     int64
	state        int32
	fsmLock      sync.Mutex
	openingTime  time.Time
}

var (
	ErrCircuitBreakerOpen error = errors.New("Circuit breaker is open")
)

type circuitBreakerOption func(*CircuitBreaker) error

const (
	stateClosed = iota
	stateOpen
	stateHalfOpen
)

func NewCircuitBreaker(options ...circuitBreakerOption) (*CircuitBreaker, error) {
	cb := &CircuitBreaker{
		circuitBreakerConfig: &circuitBreakerConfig{
			failureThreshold:    1,
			successThreshold:    1,
			openDelay:           1 * time.Minute,
			recover:             false,
			halfOpenMaxInflight: 1,
		},
		state:        stateClosed,
		failureCount: 0,
		successCount: 0,
		inflight:     0,
	}
	for _, o := range options {
		err := o(cb)
		if err != nil {
			return nil, err
		}
	}
	return cb, nil
}

func (cb *CircuitBreaker) String() string {
	inflight := atomic.LoadInt64(&cb.inflight)
	return fmt.Sprintf("cb{jobs: %v, state: %v}", inflight, cb.State())
}

func (cb *CircuitBreaker) State() string {
	state := atomic.LoadInt32(&cb.state)
	switch state {
	case stateClosed:
		return "CLOSED"
	case stateHalfOpen:
		return "HALF-OPEN"
	case stateOpen:
		return "OPEN"
	default:
		panic("must not happen")
	}
}

func FailureThreshold(failureThreshold int) circuitBreakerOption {
	return func(cb *CircuitBreaker) error {
		if failureThreshold < 1 {
			return fmt.Errorf("Failure threshold must be postive (is: %v)", failureThreshold)
		}
		cb.failureThreshold = int32(failureThreshold)
		return nil
	}
}

func SuccessThreshold(successThreshold int) circuitBreakerOption {
	return func(cb *CircuitBreaker) error {
		if successThreshold < 1 {
			return fmt.Errorf("Success threshold must be postive (is: %v)", successThreshold)
		}
		cb.successThreshold = int32(successThreshold)
		return nil
	}
}

// func OverSampleSize(sampleSize int) circuitBreakerOption {
// 	return func(cb *CircuitBreaker) error {
// 		if sampleSize < cb.failureThreshold {
// 			return fmt.Errorf("SampleSize must be >= failureThreshold (is: %v < %v)", sampleSize, cb.failureThreshold)
// 		}
// 		if sampleSize < cb.successThreshold {
// 			return fmt.Errorf("SampleSize must be >= successThreshold (is: %v < %v)", sampleSize, cb.successThreshold)
// 		}
// 		return nil
// 	}
// }

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

func (cb *CircuitBreaker) runJob(job func() error) error {
	if cb.recover {
		job = recoverDecorator(job)
	}
	err := job()
	return err
}

func (cb *CircuitBreaker) Run(job func() error) error {
	state := cb.getState()
	switch state {
	case stateOpen:
		since := time.Since(cb.openingTime)
		if since < cb.openDelay {
			return fmt.Errorf("circuit breaker is open")
		}
		cb.setState(stateHalfOpen)
		fallthrough
	case stateHalfOpen:
		err := cb.checkInflight()
		if err != nil {
			return err
		}
		fallthrough
	case stateClosed:
		cb.before()
		err := cb.runJob(job)
		cb.after(err)
		return err
	default:
		panic("Must never happen")
	}
}

func (cb *CircuitBreaker) getState() int32 {
	cb.fsmLock.Lock()
	defer cb.fsmLock.Unlock()
	return cb.state
}

func (cb *CircuitBreaker) setState(newState int32) {
	cb.fsmLock.Lock()
	defer cb.fsmLock.Unlock()
	cb.state = newState
}

func (cb *CircuitBreaker) checkInflight() error {
	cb.fsmLock.Lock()
	defer cb.fsmLock.Unlock()
	// fmt.Printf(">>> checking inflight: %d to %d\n", cb.inflight, cb.halfOpenMaxInflight)
	if cb.inflight >= cb.halfOpenMaxInflight {
		return fmt.Errorf("Error: exceeded maximum number of inflight jobs %v > %v", cb.inflight, cb.halfOpenMaxInflight)
	}
	return nil
}

func (cb *CircuitBreaker) before() {
	cb.fsmLock.Lock()
	defer cb.fsmLock.Unlock()
	// fmt.Printf(">>> before\n")
	cb.inflight++
}

func (cb *CircuitBreaker) after(err error) {
	cb.fsmLock.Lock()
	defer cb.fsmLock.Unlock()
	// fmt.Printf(">>> after\n")
	cb.inflight--
	if err == nil {
		cb.successCount++
		cb.failureCount = 0
		if cb.successCount >= cb.successThreshold && cb.state == stateHalfOpen {
			cb.state = stateClosed
		}
	} else {
		cb.failureCount++
		cb.successCount = 0
		if cb.failureCount >= cb.failureThreshold && cb.state == stateClosed {
			cb.state = stateOpen
			cb.openingTime = time.Now()
		}
	}
}
