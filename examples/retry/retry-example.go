package main

import (
	"errors"
	"time"

	"github.com/Sirupsen/logrus"
	f "github.com/adamwasila/gofailsafe/pkg"
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

func main() {
	logrus.SetLevel(logrus.DebugLevel)

	logrus.Info("#1")
	r, err := f.NewRetry()
	r.Run(alwaysWithErrorJob)

	logrus.Info("#2")
	r, err = f.NewRetry(f.Delay(100*time.Millisecond), f.Retries(15))
	r.Run(alwaysWithErrorJob)

	logrus.Info("#3")
	r, err = f.NewRetry(f.RetryIf(shouldRetry()))
	r.Run(sucessfulJob)

	logrus.Info("#4")
	r, err = f.NewRetry(f.RetryOnPanic())
	r.Run(panickingJob)

	logrus.Info("#5")
	r, err = f.NewRetry(f.Delay(-1 * time.Second))
	logrus.WithField("error", err).Info("Expected non-nil error here")

	logrus.Info("#6")
	r, err = f.NewRetry(f.Retries(-3))
	logrus.WithField("error", err).Info("Expected non-nil error here")
}
