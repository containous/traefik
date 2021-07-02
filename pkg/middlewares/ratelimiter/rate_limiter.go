// Package ratelimiter implements a rate limiting and traffic shaping middleware with a set of token buckets.
package ratelimiter

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/mailgun/ttlmap"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/middlewares"
	"github.com/traefik/traefik/v2/pkg/tracing"
	"github.com/vulcand/oxy/utils"
	"golang.org/x/time/rate"
)

const (
	typeName   = "RateLimiterType"
	maxSources = 65536
)

// rateLimiter implements rate limiting and traffic shaping with a set of token buckets;
// one for each traffic source. The same parameters are applied to all the buckets.
type rateLimiter struct {
	name  string
	rate  rate.Limit // reqs/s
	burst int64
	// maxDelay is the maximum duration we're willing to wait for a bucket reservation to become effective, in nanoseconds.
	// For now it is somewhat arbitrarily set to 1/(2*rate).
	maxDelay time.Duration
	// each rate limiter for a given source is stored in the buckets ttlmap. To keep
	// this ttlmap constrained in size, each ratelimiter is "garbage collected" when it
	// is considered expired. It is considered expired after it hasn't been used for
	// ttl seconds.
	ttl           int
	sourceMatcher utils.SourceExtractor
	next          http.Handler

	buckets *ttlmap.TtlMap // actual buckets, keyed by source.
}

// New returns a rate limiter middleware.
func New(ctx context.Context, next http.Handler, config dynamic.RateLimit, name string) (http.Handler, error) {
	ctxLog := log.With(ctx, log.Str(log.MiddlewareName, name), log.Str(log.MiddlewareType, typeName))
	log.FromContext(ctxLog).Debug("Creating middleware")

	if config.SourceCriterion == nil ||
		config.SourceCriterion.IPStrategy == nil &&
			config.SourceCriterion.RequestHeaderName == "" && !config.SourceCriterion.RequestHost {
		config.SourceCriterion = &dynamic.SourceCriterion{
			IPStrategy: &dynamic.IPStrategy{},
		}
	}

	sourceMatcher, err := middlewares.GetSourceExtractor(ctxLog, config.SourceCriterion)
	if err != nil {
		return nil, err
	}

	buckets, err := ttlmap.NewConcurrent(maxSources)
	if err != nil {
		return nil, err
	}

	burst := config.Burst
	if burst < 1 {
		burst = 1
	}

	period := time.Duration(config.Period)
	if period < 0 {
		return nil, fmt.Errorf("negative value not valid for period: %v", period)
	}
	if period == 0 {
		period = time.Second
	}

	// if config.Average == 0, the value of maxDelay does not matter since the
	// reservation will (buggily) give us a delay of 0 anyway in that case.
	var maxDelay time.Duration
	var rtl float64
	if config.Average > 0 {
		rtl = float64(config.Average*int64(time.Second)) / float64(period)
		// maxDelay does not scale well for rates below 1,
		// so we just cap it to the corresponding value, i.e. 0.5s, in order to keep the effective rate predictable.
		// One alternative would be to switch to a no-reservation mode (Allow() method) whenever we are in such a low rate regime.
		if rtl < 1 {
			maxDelay = 500 * time.Millisecond
		} else {
			maxDelay = time.Second / (time.Duration(rtl) * 2)
		}
	}

	// Make the ttl inversely proportional to how often a rate limiter is supposed
	// to see any activity (when maxed out), for low rate limiters.
	// Otherwise just make it a second for all the high rate limiters.
	// Add an extra second in both cases for continuity between the two cases.
	ttl := 1
	if rtl >= 1 {
		ttl++
	} else if rtl > 0 {
		ttl += int(1. / rtl)
	}

	return &rateLimiter{
		name:          name,
		rate:          rate.Limit(rtl),
		burst:         burst,
		maxDelay:      maxDelay,
		next:          next,
		sourceMatcher: sourceMatcher,
		buckets:       buckets,
		ttl:           ttl,
	}, nil
}

func (rl *rateLimiter) GetTracingInformation() (string, ext.SpanKindEnum) {
	return rl.name, tracing.SpanKindNoneEnum
}

func (rl *rateLimiter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := middlewares.GetLoggerCtx(r.Context(), rl.name, typeName)
	logger := log.FromContext(ctx)

	source, amount, err := rl.sourceMatcher.Extract(r)
	if err != nil {
		logger.Errorf("could not extract source of request: %v", err)
		http.Error(w, "could not extract source of request", http.StatusInternalServerError)
		return
	}

	if amount != 1 {
		logger.Infof("ignoring token bucket amount > 1: %d", amount)
	}

	var bucket *rate.Limiter
	if rlSource, exists := rl.buckets.Get(source); exists {
		bucket = rlSource.(*rate.Limiter)
	} else {
		bucket = rate.NewLimiter(rl.rate, int(rl.burst))
	}

	// We Set even in the case where the source already exists, because we want to
	// update the expiryTime everytime we get the source, as the expiryTime is
	// supposed to reflect the activity (or lack thereof) on that source.
	if err := rl.buckets.Set(source, bucket, rl.ttl); err != nil {
		logger.Errorf("could not insert/update bucket: %v", err)
		http.Error(w, "could not insert/update bucket", http.StatusInternalServerError)
		return
	}

	// time/rate is bugged imho, since a rate.Limiter with a 0 Limit not only allows
	// a Reservation to take place, but also gives a 0 delay below (because of a
	// division by zero, followed by a multiplication that flips into the negatives),
	// regardless of the current load. However, for now we take advantage of this
	// behaviour to provide the no-limit ratelimiter when config.Average is 0.
	res := bucket.Reserve()
	if !res.OK() {
		http.Error(w, "No bursty traffic allowed", http.StatusTooManyRequests)
		return
	}

	delay := res.Delay()
	if delay > rl.maxDelay {
		res.Cancel()
		rl.serveDelayError(ctx, w, r, delay)
		return
	}

	time.Sleep(delay)
	rl.next.ServeHTTP(w, r)
}

func (rl *rateLimiter) serveDelayError(ctx context.Context, w http.ResponseWriter, r *http.Request, delay time.Duration) {
	w.Header().Set("Retry-After", fmt.Sprintf("%.0f", delay.Seconds()))
	w.Header().Set("X-Retry-In", delay.String())
	w.WriteHeader(http.StatusTooManyRequests)

	if _, err := w.Write([]byte(http.StatusText(http.StatusTooManyRequests))); err != nil {
		log.FromContext(ctx).Errorf("could not serve 429: %v", err)
	}
}
