package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRateLimit(t *testing.T) {
	tests := map[string]struct {
		config           config
		requests         int
		expectedStatuses []int
	}{
		"Rate limit not exceeded": {
			config: config{
				limiter: struct {
					enabled bool
					rps     float64
					burst   int
				}{
					enabled: true,
					rps:     2,
					burst:   2,
				},
			},
			requests:         3,
			expectedStatuses: []int{200, 200, 429},
		},
		"Rate limit disabled": {
			config: config{
				limiter: struct {
					enabled bool
					rps     float64
					burst   int
				}{
					enabled: false,
					rps:     2,
					burst:   2,
				},
			},
			requests:         5,
			expectedStatuses: []int{200, 200, 200, 200, 200},
		},
		"High burst limit": {
			config: config{
				limiter: struct {
					enabled bool
					rps     float64
					burst   int
				}{
					enabled: true,
					rps:     1,
					burst:   5,
				},
			},
			requests:         6,
			expectedStatuses: []int{200, 200, 200, 200, 200, 429},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			app := &application{config: tc.config}

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			rateLimitedHandler := app.rateLimit(handler)

			for i := 0; i < tc.requests; i++ {
				req := httptest.NewRequest("GET", "/", nil)
				rr := httptest.NewRecorder()

				rateLimitedHandler.ServeHTTP(rr, req)

				assert.Equal(t, tc.expectedStatuses[i], rr.Code)

				// Add a small delay to simulate requests over time
				time.Sleep(10 * time.Millisecond)
			}
		})
	}
}
