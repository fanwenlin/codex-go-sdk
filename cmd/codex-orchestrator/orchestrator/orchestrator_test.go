package orchestrator

import (
	"errors"
	"testing"
	"unicode/utf8"
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

func TestShouldContinueTurn(t *testing.T) {
	cases := []struct {
		name     string
		response string
		want     bool
	}{
		{
			name:     "chinese follow-up cue",
			response: "预检已完成，接下来我会按清单分步调查并修复。",
			want:     true,
		},
		{
			name:     "english follow-up cue",
			response: "Precheck is done. Next I will inspect stream handling.",
			want:     true,
		},
		{
			name:     "fully completed response",
			response: "Created the requested file and finished all required steps.",
			want:     false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := shouldContinueTurn(tc.response)
			if got != tc.want {
				t.Fatalf("shouldContinueTurn(%q)=%v, want %v", tc.response, got, tc.want)
			}
		})
	}
}

func TestTruncateMaintainsValidUTF8(t *testing.T) {
	s := "先按仓库要求做会话预检，并确认环境是否正常。"
	truncated := truncate(s, 12)
	if !utf8.ValidString(truncated) {
		t.Fatalf("truncate returned invalid UTF-8: %q", truncated)
	}
	if len([]rune(truncated)) > 12 {
		t.Fatalf("truncate should keep rune length <= 12, got %d", len([]rune(truncated)))
	}
}
