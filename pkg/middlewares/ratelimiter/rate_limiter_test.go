package ratelimiter

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ptypes "github.com/traefik/paerser/types"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/testhelpers"
	"github.com/vulcand/oxy/utils"
)

func TestNewRateLimiter(t *testing.T) {
	testCases := []struct {
		desc             string
		config           dynamic.RateLimit
		expectedMaxDelay time.Duration
		expectedSourceIP string
		requestHeader    string
		expectedError    string
	}{
		{
			desc: "maxDelay computation",
			config: dynamic.RateLimit{
				Average: 200,
				Burst:   10,
			},
			expectedMaxDelay: 2500 * time.Microsecond,
		},
		{
			desc: "maxDelay computation, low rate regime",
			config: dynamic.RateLimit{
				Average: 2,
				Period:  ptypes.Duration(10 * time.Second),
				Burst:   10,
			},
			expectedMaxDelay: 500 * time.Millisecond,
		},
		{
			desc: "default SourceMatcher is remote address ip strategy",
			config: dynamic.RateLimit{
				Average: 200,
				Burst:   10,
			},
			expectedSourceIP: "127.0.0.1",
		},
		{
			desc: "SourceCriterion in config is respected",
			config: dynamic.RateLimit{
				Average: 200,
				Burst:   10,
				SourceCriterion: &dynamic.SourceCriterion{
					RequestHeaderName: "Foo",
				},
			},
			requestHeader: "bar",
		},
		{
			desc: "SourceCriteria are mutually exclusive",
			config: dynamic.RateLimit{
				Average: 200,
				Burst:   10,
				SourceCriterion: &dynamic.SourceCriterion{
					IPStrategy:        &dynamic.IPStrategy{},
					RequestHeaderName: "Foo",
				},
			},
			expectedError: "iPStrategy and RequestHeaderName are mutually exclusive",
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

			h, err := New(context.Background(), next, test.config, "rate-limiter")
			if test.expectedError != "" {
				assert.EqualError(t, err, test.expectedError)
			} else {
				require.NoError(t, err)
			}

			rtl, _ := h.(*rateLimiter)
			if test.expectedMaxDelay != 0 {
				assert.Equal(t, test.expectedMaxDelay, rtl.maxDelay)
			}

			if test.expectedSourceIP != "" {
				extractor, ok := rtl.sourceMatcher.(utils.ExtractorFunc)
				require.True(t, ok, "Not an ExtractorFunc")

				req := http.Request{
					RemoteAddr: fmt.Sprintf("%s:1234", test.expectedSourceIP),
				}

				ip, _, err := extractor(&req)
				assert.NoError(t, err)
				assert.Equal(t, test.expectedSourceIP, ip)
			}
			if test.requestHeader != "" {
				extractor, ok := rtl.sourceMatcher.(utils.ExtractorFunc)
				require.True(t, ok, "Not an ExtractorFunc")

				req := http.Request{
					Header: map[string][]string{
						test.config.SourceCriterion.RequestHeaderName: {test.requestHeader},
					},
				}
				hd, _, err := extractor(&req)
				assert.NoError(t, err)
				assert.Equal(t, test.requestHeader, hd)
			}
		})
	}
}

func TestRateLimit(t *testing.T) {
	testCases := []struct {
		desc         string
		config       dynamic.RateLimit
		loadDuration time.Duration
		incomingLoad int // in reqs/s
		burst        int
	}{
		{
			desc: "Average is respected",
			config: dynamic.RateLimit{
				Average: 100,
				Burst:   1,
			},
			loadDuration: 2 * time.Second,
			incomingLoad: 400,
		},
		{
			desc: "burst allowed, no bursty traffic",
			config: dynamic.RateLimit{
				Average: 100,
				Burst:   100,
			},
			loadDuration: 2 * time.Second,
			incomingLoad: 200,
		},
		{
			desc: "burst allowed, initial burst, under capacity",
			config: dynamic.RateLimit{
				Average: 100,
				Burst:   100,
			},
			loadDuration: 2 * time.Second,
			incomingLoad: 200,
			burst:        50,
		},
		{
			desc: "burst allowed, initial burst, over capacity",
			config: dynamic.RateLimit{
				Average: 100,
				Burst:   100,
			},
			loadDuration: 2 * time.Second,
			incomingLoad: 200,
			burst:        150,
		},
		{
			desc: "burst over average, initial burst, over capacity",
			config: dynamic.RateLimit{
				Average: 100,
				Burst:   200,
			},
			loadDuration: 2 * time.Second,
			incomingLoad: 200,
			burst:        300,
		},
		{
			desc: "lower than 1/s",
			config: dynamic.RateLimit{
				Average: 5,
				Period:  ptypes.Duration(10 * time.Second),
			},
			loadDuration: 2 * time.Second,
			incomingLoad: 100,
			burst:        0,
		},
		{
			desc: "lower than 1/s, longer",
			config: dynamic.RateLimit{
				Average: 5,
				Period:  ptypes.Duration(10 * time.Second),
			},
			loadDuration: time.Minute,
			incomingLoad: 100,
			burst:        0,
		},
		{
			desc: "lower than 1/s, longer, harsher",
			config: dynamic.RateLimit{
				Average: 1,
				Period:  ptypes.Duration(time.Minute),
			},
			loadDuration: time.Minute,
			incomingLoad: 100,
			burst:        0,
		},
		{
			desc: "period below 1 second",
			config: dynamic.RateLimit{
				Average: 50,
				Period:  ptypes.Duration(500 * time.Millisecond),
			},
			loadDuration: 2 * time.Second,
			incomingLoad: 300,
			burst:        0,
		},
		// TODO Try to disambiguate when it fails if it is because of too high a load.
		// {
		// 	desc: "Zero average ==> no rate limiting",
		// 	config: dynamic.RateLimit{
		// 		Average: 0,
		// 		Burst:   1,
		// 	},
		// 	incomingLoad: 1000,
		// 	loadDuration: time.Second,
		// },
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			if test.loadDuration >= time.Minute && testing.Short() {
				t.Skip("skipping test in short mode.")
			}
			t.Parallel()

			reqCount := 0
			dropped := 0
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				reqCount++
			})
			h, err := New(context.Background(), next, test.config, "rate-limiter")
			require.NoError(t, err)

			loadPeriod := time.Duration(1e9 / test.incomingLoad)
			start := time.Now()
			end := start.Add(test.loadDuration)
			ticker := time.NewTicker(loadPeriod)
			defer ticker.Stop()
			for {
				if time.Now().After(end) {
					break
				}

				req := testhelpers.MustNewRequest(http.MethodGet, "http://localhost", nil)
				req.RemoteAddr = "127.0.0.1:1234"
				w := httptest.NewRecorder()

				h.ServeHTTP(w, req)
				if w.Result().StatusCode != http.StatusOK {
					dropped++
				}
				if test.burst > 0 && reqCount < test.burst {
					// if a burst is defined we first hammer the server with test.burst requests as fast as possible
					continue
				}
				<-ticker.C
			}
			stop := time.Now()
			elapsed := stop.Sub(start)

			burst := test.config.Burst
			if burst < 1 {
				// actual default value
				burst = 1
			}

			period := time.Duration(test.config.Period)
			if period == 0 {
				period = time.Second
			}

			if test.config.Average == 0 {
				if reqCount < 75*test.incomingLoad/100 {
					t.Fatalf("we (arbitrarily) expect at least 75%% of the requests to go through with no rate limiting, and yet only %d/%d went through", reqCount, test.incomingLoad)
				}
				if dropped != 0 {
					t.Fatalf("no request should have been dropped if rate limiting is disabled, and yet %d were", dropped)
				}
				return
			}

			// Note that even when there is no bursty traffic,
			// we take into account the configured burst,
			// because it also helps absorbing non-bursty traffic.
			rate := float64(test.config.Average) / float64(period)

			wantCount := int(int64(rate*float64(test.loadDuration)) + burst)

			// Allow for a 2% leeway
			maxCount := wantCount * 102 / 100

			// With very high CPU loads,
			// we can expect some extra delay in addition to the rate limiting we already do,
			// so we allow for some extra leeway there.
			// Feel free to adjust wrt to the load on e.g. the CI.
			minCount := computeMinCount(wantCount)

			if reqCount < minCount {
				t.Fatalf("rate was slower than expected: %d requests (wanted > %d) in %v", reqCount, minCount, elapsed)
			}
			if reqCount > maxCount {
				t.Fatalf("rate was faster than expected: %d requests (wanted < %d) in %v", reqCount, maxCount, elapsed)
			}
		})
	}
}

func computeMinCount(wantCount int) int {
	if os.Getenv("CI") != "" {
		return wantCount * 60 / 100
	}

	return wantCount * 95 / 100
}
