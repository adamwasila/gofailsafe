package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/Sirupsen/logrus"
	failsafe "github.com/adamwasila/gofailsafe/pkg"
)

func sucessfulJob() error {
	logrus.Debug("Always succesful Job!")
	return nil
}

func alwaysWithErrorJob() error {
	logrus.Debug("Ends always with error Job!")
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

var globalCounter = 0

func errorPath(cb *failsafe.CircuitBreaker) {
	globalCounter++
	err := cb.Run(alwaysWithErrorJob)
	fmt.Printf("%d result %v\n", globalCounter, err)
}

func successPath(cb *failsafe.CircuitBreaker) {
	globalCounter++
	err := cb.Run(sucessfulJob)
	fmt.Printf("%d result %v\n", globalCounter, err)
}

func main() {
	cb, _ := failsafe.NewCircuitBreaker(failsafe.FailureThreshold(3), failsafe.SuccessThreshold(2), failsafe.OpenDelay(500*time.Millisecond))
	fmt.Printf("Circuit Breaker is: %s\n", cb.State())
	errorPath(cb)
	errorPath(cb)
	errorPath(cb)
	fmt.Printf("Circuit Breaker is: %s\n", cb.State())
	errorPath(cb)
	successPath(cb)
	errorPath(cb)
	successPath(cb)
	time.Sleep(1 * time.Second)
	fmt.Printf("Circuit Breaker is: %s\n", cb.State())
	successPath(cb)
	errorPath(cb)
	fmt.Printf("Circuit Breaker is: %s\n", cb.State())
	successPath(cb)

}
