package orchestrator

import (
	"errors"
	"testing"
)

func TestShouldFallbackToCLI(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "stream disconnected",
			err:  errors.New("thread error: stream disconnected before completion"),
			want: true,
		},
		{
			name: "app server failure",
			err:  errors.New("app server error (500): internal"),
			want: true,
		},
		{
			name: "other runtime error",
			err:  errors.New("turn failed: tool denied"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := shouldFallbackToCLI(tc.err)
			if got != tc.want {
				t.Fatalf("shouldFallbackToCLI(%v)=%v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestIsRecoverableThreadErrorMessage(t *testing.T) {
	cases := []struct {
		name    string
		message string
		want    bool
	}{
		{
			name:    "reconnecting progress error",
			message: "Reconnecting... 2/5 (stream disconnected before completion)",
			want:    true,
		},
		{
			name:    "stream retry message",
			message: "stream disconnected - retrying sampling request",
			want:    true,
		},
		{
			name:    "fatal error",
			message: "authentication failed",
			want:    false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := isRecoverableThreadErrorMessage(tc.message)
			if got != tc.want {
				t.Fatalf("isRecoverableThreadErrorMessage(%q)=%v, want %v", tc.message, got, tc.want)
			}
		})
	}
}
