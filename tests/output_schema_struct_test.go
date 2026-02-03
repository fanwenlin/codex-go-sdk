package tests

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/fanwenlin/codex-go-sdk/codex"
)

func TestOutputSchemaStructSimple(t *testing.T) {
	type SimpleOutput struct {
		Title   string  `json:"title"`
		Summary string  `json:"summary"`
		Note    *string `json:"note,omitempty"`
	}

	schema := readSchemaFromOutputSchema(t, SimpleOutput{})

	if schemaType, _ := schema["type"].(string); schemaType != "object" {
		t.Fatalf("expected root type object, got %v", schema["type"])
	}

	properties := mustMap(t, schema["properties"])
	if _, ok := properties["title"]; !ok {
		t.Fatalf("expected title property, got %v", properties)
	}
	if _, ok := properties["summary"]; !ok {
		t.Fatalf("expected summary property, got %v", properties)
	}

	required := toStringSet(t, schema["required"])
	if !required["title"] || !required["summary"] {
		t.Fatalf("expected title and summary to be required, got %v", required)
	}
	if required["note"] {
		t.Fatalf("expected note to be optional, got %v", required)
	}
}

func TestOutputSchemaStructComplex(t *testing.T) {
	type Item struct {
		Name  string  `json:"name"`
		Score float64 `json:"score"`
	}
	type Meta struct {
		Count int               `json:"count"`
		Notes map[string]string `json:"notes,omitempty"`
	}
	type ComplexOutput struct {
		ID    string   `json:"id"`
		Meta  Meta     `json:"meta"`
		Items []Item   `json:"items"`
		Tags  []string `json:"tags,omitempty"`
	}

	schema := readSchemaFromOutputSchema(t, &ComplexOutput{})
	properties := mustMap(t, schema["properties"])

	required := toStringSet(t, schema["required"])
	if !required["id"] || !required["meta"] || !required["items"] {
		t.Fatalf("expected id, meta, items to be required, got %v", required)
	}
	if required["tags"] {
		t.Fatalf("expected tags to be optional, got %v", required)
	}

	metaSchema := mustMap(t, properties["meta"])
	metaProps := mustMap(t, metaSchema["properties"])
	if _, ok := metaProps["count"]; !ok {
		t.Fatalf("expected meta.count property, got %v", metaProps)
	}

	itemsSchema := mustMap(t, properties["items"])
	if itemsSchema["type"] != "array" {
		t.Fatalf("expected items type array, got %v", itemsSchema["type"])
	}
	itemSchema := mustMap(t, itemsSchema["items"])
	itemProps := mustMap(t, itemSchema["properties"])
	if _, ok := itemProps["name"]; !ok {
		t.Fatalf("expected items.name property, got %v", itemProps)
	}
	if _, ok := itemProps["score"]; !ok {
		t.Fatalf("expected items.score property, got %v", itemProps)
	}
}

func readSchemaFromOutputSchema(t *testing.T, schema interface{}) map[string]interface{} {
	t.Helper()

	schemaFile, err := codex.CreateOutputSchemaFile(schema)
	if err != nil {
		t.Fatalf("create output schema file failed: %v", err)
	}
	t.Cleanup(func() {
		if cleanupErr := schemaFile.Cleanup(); cleanupErr != nil {
			t.Fatalf("cleanup schema file failed: %v", cleanupErr)
		}
	})

	if schemaFile.SchemaPath == "" {
		t.Fatalf("schema path should not be empty")
	}

	data, err := os.ReadFile(schemaFile.SchemaPath)
	if err != nil {
		t.Fatalf("read schema file failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal schema failed: %v", err)
	}

	return parsed
}

func mustMap(t *testing.T, value interface{}) map[string]interface{} {
	t.Helper()

	parsed, ok := value.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", value)
	}
	return parsed
}

func toStringSet(t *testing.T, value interface{}) map[string]bool {
	t.Helper()

	requiredSlice, ok := value.([]interface{})
	if !ok {
		t.Fatalf("expected required slice, got %T", value)
	}

	required := make(map[string]bool, len(requiredSlice))
	for _, item := range requiredSlice {
		if name, ok := item.(string); ok {
			required[name] = true
		}
	}
	return required
}
