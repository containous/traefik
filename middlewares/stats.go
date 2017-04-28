package middlewares

import (
	"net/http"
	"sync"
	"time"

	"github.com/containous/traefik/middlewares/common"
)

// StatsRecorder is an optional middleware that records more details statistics
// about requests and how they are processed. This currently consists of recent
// requests that have caused errors (4xx and 5xx status codes), making it easy
// to pinpoint problems.
type StatsRecorder struct {
	common.BasicMiddleware
	mutex           sync.RWMutex
	numRecentErrors int
	recentErrors    []*statsError
}

var _ common.Middleware = &StatsRecorder{}

// NewStatsRecorder returns a new StatsRecorder
func NewStatsRecorder(numRecentErrors int, next http.Handler) common.Middleware {
	return &StatsRecorder{
		BasicMiddleware: common.NewMiddleware(next),
		numRecentErrors: numRecentErrors,
	}
}

// Stats includes all of the stats gathered by the recorder.
type Stats struct {
	RecentErrors []*statsError `json:"recent_errors"`
}

// statsError represents an error that has occurred during request processing.
type statsError struct {
	StatusCode int       `json:"status_code"`
	Status     string    `json:"status"`
	Method     string    `json:"method"`
	Host       string    `json:"host"`
	Path       string    `json:"path"`
	Time       time.Time `json:"time"`
}

// responseRecorder captures information from the response and preserves it for
// later analysis.
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code for later retrieval.
func (r *responseRecorder) WriteHeader(status int) {
	r.ResponseWriter.WriteHeader(status)
	r.statusCode = status
}

// ServeHTTP silently extracts information from the request and response as it
// is processed. If the response is 4xx or 5xx, add it to the list of 10 most
// recent errors.
func (s *StatsRecorder) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	recorder := &responseRecorder{w, http.StatusOK}
	s.Next().ServeHTTP(recorder, r)

	if recorder.statusCode >= 400 {
		s.mutex.Lock()
		defer s.mutex.Unlock()
		s.recentErrors = append([]*statsError{
			{
				StatusCode: recorder.statusCode,
				Status:     http.StatusText(recorder.statusCode),
				Method:     r.Method,
				Host:       r.Host,
				Path:       r.URL.Path,
				Time:       time.Now(),
			},
		}, s.recentErrors...)
		// Limit the size of the list to numRecentErrors
		if len(s.recentErrors) > s.numRecentErrors {
			s.recentErrors = s.recentErrors[:s.numRecentErrors]
		}
	}
}

// Data returns a copy of the statistics that have been gathered.
func (s *StatsRecorder) Data() *Stats {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// We can't return the slice directly or a race condition might develop
	recentErrors := make([]*statsError, len(s.recentErrors))
	copy(recentErrors, s.recentErrors)

	return &Stats{
		RecentErrors: recentErrors,
	}
}
