package tests

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"strings"
	"testing"
)

//go:embed testdata/verbose_sanitized.jsonl
var verboseJSONL string

func TestVerboseJSONLParses(t *testing.T) {
	if strings.TrimSpace(verboseJSONL) == "" {
		t.Fatal("embedded verbose JSONL is empty")
	}

	scanner := bufio.NewScanner(strings.NewReader(verboseJSONL))
	scanner.Buffer(make([]byte, 1024), 2*1024*1024)

	lines := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var payload map[string]interface{}
		if err := json.Unmarshal([]byte(line), &payload); err != nil {
			t.Fatalf("failed to parse line %d: %v", lines+1, err)
		}
		if len(payload) == 0 {
			t.Fatalf("line %d parsed to empty object", lines+1)
		}
		lines++
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanner error: %v", err)
	}
	if lines == 0 {
		t.Fatal("no JSONL lines parsed")
	}
}
