package codex

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// OutputSchemaFile represents a temporary output schema file.
type OutputSchemaFile struct {
	// SchemaPath is the path to the schema file
	SchemaPath string
	// Cleanup is a function to clean up the temporary file
	Cleanup func() error
}

// CreateOutputSchemaFile creates a temporary JSON schema file.
// If schema is nil, it returns an empty OutputSchemaFile with a no-op cleanup function.
// The schema must be a valid JSON object (not an array or primitive).
func CreateOutputSchemaFile(schema interface{}) (*OutputSchemaFile, error) {
	if schema == nil {
		return &OutputSchemaFile{
			Cleanup: func() error { return nil },
		}, nil
	}

	// Check if schema is a valid JSON object (not an array or primitive)
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}

	var tempMap map[string]interface{}
	if unmarshalErr := json.Unmarshal(schemaBytes, &tempMap); unmarshalErr != nil {
		return nil, unmarshalErr
	}

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "codex-output-schema-")
	if err != nil {
		return nil, err
	}

	schemaPath := filepath.Join(tempDir, "schema.json")

	// Write schema to file
	if writeErr := os.WriteFile(schemaPath, schemaBytes, 0600); writeErr != nil {
		_ = os.RemoveAll(tempDir)
		return nil, writeErr
	}

	// Create cleanup function
	cleanup := func() error {
		return os.RemoveAll(tempDir)
	}

	return &OutputSchemaFile{
		SchemaPath: schemaPath,
		Cleanup:    cleanup,
	}, nil
}
