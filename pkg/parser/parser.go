// Package parser handles parsing workflow definitions from JSON and YAML formats.
package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/JSLEEKR/flowlint/pkg/dag"
	"gopkg.in/yaml.v3"
)

// Format represents a supported workflow file format.
type Format string

const (
	FormatJSON    Format = "json"
	FormatYAML    Format = "yaml"
	FormatUnknown Format = "unknown"
)

// DetectFormat determines the file format from the file extension.
func DetectFormat(filename string) Format {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".json":
		return FormatJSON
	case ".yaml", ".yml":
		return FormatYAML
	default:
		return FormatUnknown
	}
}

// MaxFileSize is the maximum allowed workflow file size (10 MB).
const MaxFileSize = 10 * 1024 * 1024

// ParseFile reads and parses a workflow file, auto-detecting format from extension.
func ParseFile(path string) (*dag.Workflow, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("reading file %q: %w", path, err)
	}
	if info.Size() > MaxFileSize {
		return nil, fmt.Errorf("file %q is too large (%d bytes, max %d)", path, info.Size(), MaxFileSize)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file %q: %w", path, err)
	}

	format := DetectFormat(path)
	if format == FormatUnknown {
		return nil, fmt.Errorf("unsupported file format for %q: use .json, .yaml, or .yml", path)
	}

	return ParseBytes(data, format)
}

// ParseBytes parses workflow data from bytes in the specified format.
func ParseBytes(data []byte, format Format) (*dag.Workflow, error) {
	var w dag.Workflow

	switch format {
	case FormatJSON:
		if err := json.Unmarshal(data, &w); err != nil {
			return nil, fmt.Errorf("parsing JSON: %w", err)
		}
	case FormatYAML:
		if err := yaml.Unmarshal(data, &w); err != nil {
			return nil, fmt.Errorf("parsing YAML: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported format: %q", format)
	}

	if len(w.Steps) == 0 {
		return nil, fmt.Errorf("workflow has no steps defined")
	}

	// Assign default type if not specified
	for i := range w.Steps {
		if w.Steps[i].Type == "" {
			w.Steps[i].Type = dag.StepTypeTask
		}
	}

	return &w, nil
}
