package try

import (
	"fmt"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/traefik/traefik/v2/pkg/log"
)

const (
	// CITimeoutMultiplier is the multiplier for all timeout in the CI.
	CITimeoutMultiplier = 3
	maxInterval         = 5 * time.Second
)

type timedAction func(timeout time.Duration, operation DoCondition) error

// Sleep pauses the current goroutine for at least the duration d.
// Deprecated: Use only when use an other Try[...] functions is not possible.
func Sleep(d time.Duration) {
	d = applyCIMultiplier(d)
	time.Sleep(d)
}

// Response is like Request, but returns the response for further
// processing at the call site.
// Conditions are not allowed since it would complicate signaling if the
// response body needs to be closed or not. Callers are expected to close on
// their own if the function returns a nil error.
func Response(req *http.Request, timeout time.Duration) (*http.Response, error) {
	return doTryRequest(req, timeout, nil)
}

// ResponseUntilStatusCode is like Request, but returns the response for further
// processing at the call site.
// Conditions are not allowed since it would complicate signaling if the
// response body needs to be closed or not. Callers are expected to close on
// their own if the function returns a nil error.
func ResponseUntilStatusCode(req *http.Request, timeout time.Duration, statusCode int) (*http.Response, error) {
	return doTryRequest(req, timeout, nil, StatusCodeIs(statusCode))
}

// GetRequest is like Do, but runs a request against the given URL and applies
// the condition on the response.
// ResponseCondition may be nil, in which case only the request against the URL must
// succeed.
func GetRequest(url string, timeout time.Duration, conditions ...ResponseCondition) error {
	resp, err := doTryGet(url, timeout, nil, conditions...)

	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	return err
}

// Request is like Do, but runs a request against the given URL and applies
// the condition on the response.
// ResponseCondition may be nil, in which case only the request against the URL must
// succeed.
func Request(req *http.Request, timeout time.Duration, conditions ...ResponseCondition) error {
	resp, err := doTryRequest(req, timeout, nil, conditions...)

	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	return err
}

// RequestWithTransport is like Do, but runs a request against the given URL and applies
// the condition on the response.
// ResponseCondition may be nil, in which case only the request against the URL must
// succeed.
func RequestWithTransport(req *http.Request, timeout time.Duration, transport *http.Transport, conditions ...ResponseCondition) error {
	resp, err := doTryRequest(req, timeout, transport, conditions...)

	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	return err
}

// Do repeatedly executes an operation until no error condition occurs or the
// given timeout is reached, whatever comes first.
func Do(timeout time.Duration, operation DoCondition) error {
	if timeout <= 0 {
		panic("timeout must be larger than zero")
	}

	interval := time.Duration(math.Ceil(float64(timeout) / 15.0))
	if interval > maxInterval {
		interval = maxInterval
	}

	timeout = applyCIMultiplier(timeout)

	var err error
	if err = operation(); err == nil {
		fmt.Println("+")
		return nil
	}
	fmt.Print("*")

	stopTimer := time.NewTimer(timeout)
	defer stopTimer.Stop()
	retryTick := time.NewTicker(interval)
	defer retryTick.Stop()

	for {
		select {
		case <-stopTimer.C:
			fmt.Println("-")
			return fmt.Errorf("try operation failed: %w", err)
		case <-retryTick.C:
			fmt.Print("*")
			if err = operation(); err == nil {
				fmt.Println("+")
				return nil
			}
		}
	}
}

func doTryGet(url string, timeout time.Duration, transport http.RoundTripper, conditions ...ResponseCondition) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	return doTryRequest(req, timeout, transport, conditions...)
}

func doTryRequest(request *http.Request, timeout time.Duration, transport http.RoundTripper, conditions ...ResponseCondition) (*http.Response, error) {
	return doRequest(Do, timeout, request, transport, conditions...)
}

func doRequest(action timedAction, timeout time.Duration, request *http.Request, transport http.RoundTripper, conditions ...ResponseCondition) (*http.Response, error) {
	var resp *http.Response
	return resp, action(timeout, func() error {
		var err error
		client := http.DefaultClient
		if transport != nil {
			client.Transport = transport
		}

		resp, err = client.Do(request)
		if err != nil {
			return err
		}

		for _, condition := range conditions {
			if err := condition(resp); err != nil {
				return err
			}
		}

		return nil
	})
}

func applyCIMultiplier(timeout time.Duration) time.Duration {
	ci := os.Getenv("CI")
	if len(ci) > 0 {
		log.Debug("Apply CI multiplier:", CITimeoutMultiplier)
		return time.Duration(float64(timeout) * CITimeoutMultiplier)
	}
	return timeout
}
