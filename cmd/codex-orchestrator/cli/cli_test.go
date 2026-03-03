package cli

import "testing"

func TestParseArgsQuietValid(t *testing.T) {
	options, errs := parseArgs([]string{"docs", "--quiet"})
	if len(errs) > 0 {
		t.Fatalf("parseArgs returned errors: %v", errs)
	}
	if !options.Quiet {
		t.Fatalf("options.Quiet=%v, want true", options.Quiet)
	}
}

func TestParseArgsUnknownFlagInvalid(t *testing.T) {
	_, errs := parseArgs([]string{"docs", "--max-turns", "3"})
	if len(errs) == 0 {
		t.Fatalf("expected parseArgs to return errors for unknown flag")
	}
}
