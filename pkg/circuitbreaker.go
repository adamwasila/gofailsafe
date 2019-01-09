package failsafe

import (
	"fmt"
	"sync/atomic"
	"time"
)

type circuitBreakerConfig struct {
	failureThreshold    int
	successThreshold    int
	openDelay           time.Duration
	recover             bool
	halfOpenMaxInflight int64
}

type CircuitBreaker struct {
	*circuitBreakerConfig
	executions CircularBoolArray
	inflight   int64
	state      int32
	ex         chan bool
	quit       chan struct{}
}

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
		executions: *NewCircularBoolArray(1),
		inflight:   0,
		state:      stateClosed,
		ex:         make(chan bool),
		quit:       make(chan struct{}),
	}
	for _, o := range options {
		err := o(cb)
		if err != nil {
			return nil, err
		}
	}
	go cb.startMonitor()
	return cb, nil
}

func (cb *CircuitBreaker) Close() {
	close(cb.quit)
}

func (cb *CircuitBreaker) String() string {
	inflight := atomic.LoadInt64(&cb.inflight)
	return fmt.Sprintf("cb{jobs: %v, state: %v, ex: %v}", inflight, cb.State(), cb.executions.Count())
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
		cb.failureThreshold = failureThreshold
		return nil
	}
}

func SuccessThreshold(successThreshold int) circuitBreakerOption {
	return func(cb *CircuitBreaker) error {
		if successThreshold < 1 {
			return fmt.Errorf("Success threshold must be postive (is: %v)", successThreshold)
		}
		cb.successThreshold = successThreshold
		return nil
	}
}

func OverSampleSize(sampleSize int) circuitBreakerOption {
	return func(cb *CircuitBreaker) error {
		if sampleSize < cb.failureThreshold {
			return fmt.Errorf("SampleSize must be >= failureThreshold (is: %v < %v)", sampleSize, cb.failureThreshold)
		}
		if sampleSize < cb.successThreshold {
			return fmt.Errorf("SampleSize must be >= successThreshold (is: %v < %v)", sampleSize, cb.successThreshold)
		}
		cb.executions = *NewCircularBoolArray(sampleSize)
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

func (cb *CircuitBreaker) runJob(job func() error, finishedCallback func(error)) error {
	if cb.recover {
		job = recoverDecorator(job)
	}
	atomic.AddInt64(&cb.inflight, 1)
	err := job()
	atomic.AddInt64(&cb.inflight, -1)
	finishedCallback(err)
	return err
}

func (cb *CircuitBreaker) Run(job func() error) error {
	if cb.recover {
		job = recoverDecorator(job)
	}
	inFlight := atomic.LoadInt64(&cb.inflight)
	state := atomic.LoadInt32(&cb.state)
	switch state {
	case stateHalfOpen:
		if inFlight >= cb.halfOpenMaxInflight {
			return fmt.Errorf("Error: exceeded maximum number of inflight jobs %v > %v", inFlight, cb.halfOpenMaxInflight)
		}
		fallthrough
	case stateClosed:
		return cb.runJob(job, func(err error) {
			cb.ex <- err == nil
		})
	case stateOpen:
		return fmt.Errorf("Error: circuit breaker is open")
	default:
		panic("Must never happen")
	}
}

func (cb *CircuitBreaker) startMonitor() {
	for {
		select {
		case exStatus := <-cb.ex:
			cb.executions.Insert(exStatus)
			switch cb.state {
			case stateClosed:
				if cb.executions.CountFalse() >= cb.failureThreshold {
					fmt.Println("Opening circuit breaker!")
					cb.executions.Reset(false)
					atomic.StoreInt32(&cb.state, stateOpen)
					time.AfterFunc(cb.openDelay, func() {
						cb.executions.Reset(false)
						atomic.StoreInt32(&cb.state, stateHalfOpen)
					})
				}
			case stateHalfOpen:
				cb.executions.Insert(exStatus)
				if cb.executions.Count() >= cb.successThreshold {
					fmt.Println("Closing circuit breaker!")
					cb.executions.Reset(true)
					atomic.StoreInt32(&cb.state, stateClosed)
				}
			case stateOpen:
				break
			default:
				panic("Must never happen")
			}

		case <-cb.quit:
			fmt.Printf("Quit monitoring")
			return
		}
	}
}
