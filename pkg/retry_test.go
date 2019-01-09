package failsafe_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
	failsafe "github.com/adamwasila/gofailsafe/pkg"
	"github.com/stretchr/testify/assert"
)

func TestEmptyRetry(t *testing.T) {
	_, err := failsafe.NewRetry()
	assert.Nil(t, err, "Empty, no-op retry must be possible to create")
}

func TestRetryWithCustomDelay(t *testing.T) {
	_, err := failsafe.NewRetry(failsafe.Delay(100 * time.Second))
	assert.Nil(t, err, "Retry with custom delay must be possible to create")
}

func TestRetryWithCustomInvalidDelay(t *testing.T) {
	_, err := failsafe.NewRetry(failsafe.Delay(-1 * time.Second))
	assert.NotNil(t, err, "Retry with custom, negative delay must return error")
}

func TestRetryWithCustomRetries(t *testing.T) {
	_, err := failsafe.NewRetry(failsafe.Retries(1))
	assert.Nil(t, err, "Retry with custom number of retries must be possible to create")
}

func TestRetryWithCustomInvalidRetries(t *testing.T) {
	_, err := failsafe.NewRetry(failsafe.Retries(-1))
	assert.NotNil(t, err, "Retry with custom, negative number of retries must return error")
}

func nilTask() error {
	return errors.New("please continue with retry")
}

func TestDelay(t *testing.T) {
	r, err := failsafe.NewRetry(failsafe.Retries(2), failsafe.Delay(100*time.Millisecond))
	assert.Nil(t, err)
	start := time.Now()
	r.Run(nilTask)
	elapsed := time.Since(start)
	assert.InDelta(t, int(100*time.Millisecond), int(elapsed), float64(10*time.Millisecond))
}

func TestDelay2(t *testing.T) {
	r, err := failsafe.NewRetry(failsafe.Retries(7), failsafe.RetryIf(
		func(result interface{}, err error) bool {
			return result.(int) < 15
		},
	))
	assert.Nil(t, err)
	i := 0
	r.Get(func() (interface{}, error) {
		logrus.Info("called")
		i++
		return i, nil
	})
	t.Logf("i: %d", i)
}
