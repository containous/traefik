package job

import (
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
)

func TestJobBackOff(t *testing.T) {
	var (
		testInitialInterval     = 500 * time.Millisecond
		testRandomizationFactor = 0.1
		testMultiplier          = 2.0
		testMaxInterval         = 5 * time.Second
		testMinJobInterval      = 1 * time.Second
	)

	exp := NewBackOff(backoff.NewExponentialBackOff())
	exp.InitialInterval = testInitialInterval
	exp.RandomizationFactor = testRandomizationFactor
	exp.Multiplier = testMultiplier
	exp.MaxInterval = testMaxInterval
	exp.MinJobInterval = testMinJobInterval
	exp.Reset()

	expectedResults := []time.Duration{
		500 * time.Millisecond,
		500 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
		2 * time.Second,
		4 * time.Second,
		5 * time.Second,
		5 * time.Second,
		500 * time.Millisecond,
		1 * time.Second,
		2 * time.Second,
		4 * time.Second,
		5 * time.Second,
		5 * time.Second,
	}

	for i, expected := range expectedResults {
		// Assert that the next backoff falls in the expected range.
		minInterval := expected - time.Duration(testRandomizationFactor*float64(expected))
		maxInterval := expected + time.Duration(testRandomizationFactor*float64(expected))

		if i < 3 || i == 8 {
			time.Sleep(2 * time.Second)
		}

		actualInterval := exp.NextBackOff()
		if !(minInterval <= actualInterval && actualInterval <= maxInterval) {
			t.Error("error")
		}
	}
}
