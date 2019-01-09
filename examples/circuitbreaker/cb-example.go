package main

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/Sirupsen/logrus"
	failsafe "github.com/adamwasila/gofailsafe/pkg"
)

func sucessfulJob() error {
	logrus.Debug("Always succesful Job!")
	time.Sleep(1 * time.Second)
	logrus.Debug("Always succesful Job ended!")
	return nil
}

func alwaysWithErrorJob() error {
	logrus.Debug("Ends always with error Job!")
	time.Sleep(1 * time.Second)
	return errors.New("Doh!")
}

func panickingJob() error {
	logrus.Debug("Panicking Job!")
	panic("oh no!")
}

func shouldRetry() func(res interface{}, err error) bool {
	i := 0
	return func(res interface{}, err error) bool {
		i++
		return i < 2
	}
}

var globalCounter int64

func errorPath(cb *failsafe.CircuitBreaker) {
	atomic.AddInt64(&globalCounter, 1)
	cb.Run(alwaysWithErrorJob)
	// fmt.Printf("%d result %v\n", atomic.LoadInt64(&globalCounter), err)
}

func successPath(cb *failsafe.CircuitBreaker) {
	atomic.AddInt64(&globalCounter, 1)
	cb.Run(sucessfulJob)
	// fmt.Printf("%d result %v\n", atomic.LoadInt64(&globalCounter), err)
}

func main() {
	logrus.SetLevel(logrus.InfoLevel)
	cb, _ := failsafe.NewCircuitBreaker(failsafe.FailureThreshold(3), failsafe.SuccessThreshold(2), failsafe.OpenDelay(3000*time.Millisecond))
	for i := 0; i < 10; i++ {
		go successPath(cb)
	}
	time.Sleep(100 * time.Millisecond)
	fmt.Printf("Circuit Breaker: %v\n", cb)
	time.Sleep(1000 * time.Millisecond)
	fmt.Printf("Circuit Breaker: %v\n", cb)
	for i := 0; i < 3; i++ {
		go errorPath(cb)
	}
	time.Sleep(100 * time.Millisecond)
	fmt.Printf("Circuit Breaker: %v\n", cb)
	fmt.Printf("Sleeping 3s before exit")
	time.Sleep(3 * time.Second)
}
