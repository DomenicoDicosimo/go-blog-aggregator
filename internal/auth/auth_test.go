package auth

import (
	"errors"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGetAPIKey(t *testing.T) {
	tests := map[string]struct {
		headers       http.Header
		expectedKey   string
		expectedError error
	}{
		"No authorization header included": {
			headers:       http.Header{},
			expectedKey:   "",
			expectedError: ErrNoAuthHeaderIncluded,
		},
		"Malformed authorization header - Bearer token": {
			headers:       http.Header{"Authorization": {"Bearer some-token"}},
			expectedKey:   "",
			expectedError: errors.New("malformed authorization header"),
		},
		"Valid ApiKey authorization header": {
			headers:       http.Header{"Authorization": {"ApiKey my-api-key"}},
			expectedKey:   "my-api-key",
			expectedError: nil,
		},
		"Invalid ApiKey authorization header format": {
			headers:       http.Header{"Authorization": {"ApiKey"}},
			expectedKey:   "",
			expectedError: errors.New("malformed authorization header"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			gotKey, gotErr := GetBearerToken(tc.headers)

			if diff := cmp.Diff(tc.expectedKey, gotKey); diff != "" {
				t.Errorf("Key mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.expectedError, gotErr, cmp.Comparer(func(x, y error) bool {
				if x == nil && y == nil {
					return true
				}
				if x == nil || y == nil {
					return false
				}
				return x.Error() == y.Error()
			})); diff != "" {
				t.Errorf("Error mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
