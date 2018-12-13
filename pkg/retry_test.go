package failsafe_test

import (
	"testing"
	"time"

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
