package main

import (
	"testing"
)

func TestValidateUserParams(t *testing.T) {
	tests := map[string]struct {
		params  userParameters
		wantErr bool
	}{
		"valid params": {
			params:  userParameters{Name: "Johnny Appleseed"},
			wantErr: false,
		},
		"name too short": {
			params:  userParameters{Name: "J"},
			wantErr: true,
		},
		"name too long": {
			params:  userParameters{"A very long name that exceeds the maximum allowed length of one hundred characters and then some more..."},
			wantErr: true,
		},
		"empty name": {
			params:  userParameters{Name: ""},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateUserParams(tc.params)
			if (err != nil) != tc.wantErr {
				t.Fatalf("expected error: %v, got: %v", tc.wantErr, err != nil)
			}
		})
	}
}

func TestValidateFeedParams(t *testing.T) {
	tests := map[string]struct {
		params  feedParameters
		wantErr bool
	}{
		"valid params": {
			params:  feedParameters{Name: "Johnny Appleseed", URL: "https://example.com"},
			wantErr: false,
		},
		"url too short": {
			params:  feedParameters{Name: "Johnny Appleseed", URL: "htt"},
			wantErr: true,
		},
		"empty url": {
			params:  feedParameters{Name: "Johnny Appleseed", URL: ""},
			wantErr: true,
		},
		"invalid url": {
			params:  feedParameters{Name: "Johnny Appleseed", URL: "invalid-url"},
			wantErr: true,
		},
	}

	for Url, tc := range tests {
		t.Run(Url, func(t *testing.T) {
			err := validateFeedParams(tc.params)
			if (err != nil) != tc.wantErr {
				t.Fatalf("expected error: %v, got: %v", tc.wantErr, err != nil)
			}
		})
	}
}
