package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectFormat_JSON(t *testing.T) {
	if got := DetectFormat("workflow.json"); got != FormatJSON {
		t.Errorf("expected json, got %v", got)
	}
}

func TestDetectFormat_YAML(t *testing.T) {
	if got := DetectFormat("workflow.yaml"); got != FormatYAML {
		t.Errorf("expected yaml, got %v", got)
	}
	if got := DetectFormat("workflow.yml"); got != FormatYAML {
		t.Errorf("expected yaml for .yml, got %v", got)
	}
}

func TestDetectFormat_Unknown(t *testing.T) {
	if got := DetectFormat("workflow.txt"); got != FormatUnknown {
		t.Errorf("expected unknown, got %v", got)
	}
}

func TestDetectFormat_CaseInsensitive(t *testing.T) {
	if got := DetectFormat("workflow.JSON"); got != FormatJSON {
		t.Errorf("expected json, got %v", got)
	}
	if got := DetectFormat("workflow.YAML"); got != FormatYAML {
		t.Errorf("expected yaml, got %v", got)
	}
}

func TestParseBytes_JSON(t *testing.T) {
	data := []byte(`{
		"name": "test",
		"steps": [
			{"id": "step1"},
			{"id": "step2", "depends_on": ["step1"]}
		]
	}`)
	w, err := ParseBytes(data, FormatJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Name != "test" {
		t.Errorf("expected name 'test', got %q", w.Name)
	}
	if len(w.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(w.Steps))
	}
}

func TestParseBytes_YAML(t *testing.T) {
	data := []byte(`
name: test
steps:
  - id: step1
  - id: step2
    depends_on:
      - step1
`)
	w, err := ParseBytes(data, FormatYAML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Name != "test" {
		t.Errorf("expected name 'test', got %q", w.Name)
	}
	if len(w.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(w.Steps))
	}
}

func TestParseBytes_EmptySteps(t *testing.T) {
	data := []byte(`{"name": "empty", "steps": []}`)
	_, err := ParseBytes(data, FormatJSON)
	if err == nil {
		t.Fatal("expected error for empty steps")
	}
}

func TestParseBytes_InvalidJSON(t *testing.T) {
	data := []byte(`{invalid json}`)
	_, err := ParseBytes(data, FormatJSON)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseBytes_InvalidYAML(t *testing.T) {
	data := []byte(":\n  :\n    - [invalid")
	_, err := ParseBytes(data, FormatYAML)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestParseBytes_UnsupportedFormat(t *testing.T) {
	_, err := ParseBytes([]byte("test"), FormatUnknown)
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}
}

func TestParseBytes_DefaultStepType(t *testing.T) {
	data := []byte(`{"steps": [{"id": "a"}]}`)
	w, err := ParseBytes(data, FormatJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Steps[0].Type != "task" {
		t.Errorf("expected default type 'task', got %q", w.Steps[0].Type)
	}
}

func TestParseBytes_WithTypeInfo(t *testing.T) {
	data := []byte(`{
		"steps": [{
			"id": "a",
			"input": {"name": "data", "format": "json"},
			"output": {"name": "result", "format": "string"}
		}]
	}`)
	w, err := ParseBytes(data, FormatJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Steps[0].Input == nil || w.Steps[0].Input.Format != "json" {
		t.Errorf("expected input format 'json'")
	}
	if w.Steps[0].Output == nil || w.Steps[0].Output.Format != "string" {
		t.Errorf("expected output format 'string'")
	}
}

func TestParseBytes_WithErrorHandler(t *testing.T) {
	data := []byte(`{
		"steps": [{
			"id": "a",
			"error_handler": {
				"strategy": "retry",
				"max_retry": 3,
				"timeout": "30s"
			}
		}]
	}`)
	w, err := ParseBytes(data, FormatJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Steps[0].ErrorHandler == nil {
		t.Fatal("expected error handler")
	}
	if w.Steps[0].ErrorHandler.Strategy != "retry" {
		t.Errorf("expected strategy 'retry', got %q", w.Steps[0].ErrorHandler.Strategy)
	}
	if w.Steps[0].ErrorHandler.MaxRetry != 3 {
		t.Errorf("expected max_retry 3, got %d", w.Steps[0].ErrorHandler.MaxRetry)
	}
}

func TestParseFile_JSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "workflow.json")
	data := []byte(`{"steps": [{"id": "a"}, {"id": "b", "depends_on": ["a"]}]}`)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	w, err := ParseFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(w.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(w.Steps))
	}
}

func TestParseFile_YAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "workflow.yaml")
	data := []byte("steps:\n  - id: a\n  - id: b\n    depends_on:\n      - a\n")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	w, err := ParseFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(w.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(w.Steps))
	}
}

func TestParseFile_NotFound(t *testing.T) {
	_, err := ParseFile("/nonexistent/file.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestParseFile_UnsupportedExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "workflow.txt")
	if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	_, err := ParseFile(path)
	if err == nil {
		t.Fatal("expected error for unsupported extension")
	}
}

func TestParseBytes_YAML_WithAllFields(t *testing.T) {
	data := []byte(`
name: full-workflow
version: "2.0"
description: A complete workflow
steps:
  - id: fetch
    name: Fetch Data
    type: task
    timeout: "60s"
    retry: 3
    output:
      name: raw_data
      format: json
    error_handler:
      strategy: retry
      max_retry: 3
      timeout: "10s"
  - id: process
    name: Process Data
    type: task
    depends_on:
      - fetch
    input:
      name: raw_data
      format: json
    output:
      name: result
      format: string
`)
	w, err := ParseBytes(data, FormatYAML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Name != "full-workflow" {
		t.Errorf("expected name 'full-workflow', got %q", w.Name)
	}
	if w.Version != "2.0" {
		t.Errorf("expected version '2.0', got %q", w.Version)
	}
	if len(w.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(w.Steps))
	}
	if w.Steps[0].Timeout != "60s" {
		t.Errorf("expected timeout '60s', got %q", w.Steps[0].Timeout)
	}
	if w.Steps[0].Retry != 3 {
		t.Errorf("expected retry 3, got %d", w.Steps[0].Retry)
	}
}
