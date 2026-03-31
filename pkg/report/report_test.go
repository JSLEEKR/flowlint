package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/JSLEEKR/flowlint/pkg/lint"
)

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input   string
		want    Format
		wantErr bool
	}{
		{"text", FormatText, false},
		{"", FormatText, false},
		{"json", FormatJSON, false},
		{"sarif", FormatSARIF, false},
		{"TEXT", FormatText, false},
		{"JSON", FormatJSON, false},
		{"SARIF", FormatSARIF, false},
		{"xml", "", true},
		{"csv", "", true},
	}
	for _, tt := range tests {
		got, err := ParseFormat(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseFormat(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
		}
		if got != tt.want {
			t.Errorf("ParseFormat(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestWriteText_NoFindings(t *testing.T) {
	var buf bytes.Buffer
	err := Write(&buf, nil, FormatText, "test.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "no issues found") {
		t.Errorf("expected 'no issues found', got %q", buf.String())
	}
}

func TestWriteText_WithFindings(t *testing.T) {
	findings := []lint.Finding{
		{RuleID: "DAG001", Severity: lint.SeverityError, Message: "cycle detected", StepID: "a"},
		{RuleID: "ERR001", Severity: lint.SeverityWarning, Message: "no handler", StepID: "b"},
	}
	var buf bytes.Buffer
	err := Write(&buf, findings, FormatText, "test.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "E error") {
		t.Errorf("expected error indicator, got %q", output)
	}
	if !strings.Contains(output, "W warning") {
		t.Errorf("expected warning indicator, got %q", output)
	}
	if !strings.Contains(output, "[step: a]") {
		t.Errorf("expected step info, got %q", output)
	}
}

func TestWriteText_WithStepIDs(t *testing.T) {
	findings := []lint.Finding{
		{RuleID: "DAG001", Severity: lint.SeverityError, Message: "cycle", StepIDs: []string{"a", "b"}},
	}
	var buf bytes.Buffer
	err := Write(&buf, findings, FormatText, "test.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "[steps: a, b]") {
		t.Errorf("expected steps info, got %q", buf.String())
	}
}

func TestWriteJSON(t *testing.T) {
	findings := []lint.Finding{
		{RuleID: "DAG001", Severity: lint.SeverityError, Message: "test"},
	}
	var buf bytes.Buffer
	err := Write(&buf, findings, FormatJSON, "test.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result JSONReport
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if len(result.Findings) != 1 {
		t.Errorf("expected 1 finding, got %d", len(result.Findings))
	}
	if result.File != "test.yaml" {
		t.Errorf("expected file 'test.yaml', got %q", result.File)
	}
	if result.Summary == "" {
		t.Error("expected non-empty summary")
	}
}

func TestWriteJSON_Empty(t *testing.T) {
	var buf bytes.Buffer
	err := Write(&buf, nil, FormatJSON, "test.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result JSONReport
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if result.File != "test.yaml" {
		t.Errorf("expected file 'test.yaml', got %q", result.File)
	}
	if len(result.Findings) != 0 {
		t.Errorf("expected 0 findings, got %d", len(result.Findings))
	}
}

func TestWriteSARIF(t *testing.T) {
	findings := []lint.Finding{
		{RuleID: "DAG001", Severity: lint.SeverityError, Message: "cycle detected"},
		{RuleID: "ERR001", Severity: lint.SeverityWarning, Message: "no handler"},
	}
	var buf bytes.Buffer
	err := Write(&buf, findings, FormatSARIF, "test.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var report sarifReport
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("invalid SARIF output: %v", err)
	}

	if report.Version != "2.1.0" {
		t.Errorf("expected SARIF version 2.1.0, got %q", report.Version)
	}
	if len(report.Runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(report.Runs))
	}
	if report.Runs[0].Tool.Driver.Name != "flowlint" {
		t.Errorf("expected tool name 'flowlint', got %q", report.Runs[0].Tool.Driver.Name)
	}
	if len(report.Runs[0].Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(report.Runs[0].Results))
	}
	if len(report.Runs[0].Tool.Driver.Rules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(report.Runs[0].Tool.Driver.Rules))
	}
}

func TestWriteSARIF_Levels(t *testing.T) {
	findings := []lint.Finding{
		{RuleID: "R1", Severity: lint.SeverityError, Message: "e"},
		{RuleID: "R2", Severity: lint.SeverityWarning, Message: "w"},
		{RuleID: "R3", Severity: lint.SeverityInfo, Message: "i"},
	}
	var buf bytes.Buffer
	err := Write(&buf, findings, FormatSARIF, "test.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var report sarifReport
	json.Unmarshal(buf.Bytes(), &report)

	expectedLevels := []string{"error", "warning", "note"}
	for i, result := range report.Runs[0].Results {
		if result.Level != expectedLevels[i] {
			t.Errorf("result %d: expected level %q, got %q", i, expectedLevels[i], result.Level)
		}
	}
}

func TestWriteSARIF_WithLocation(t *testing.T) {
	findings := []lint.Finding{
		{RuleID: "R1", Severity: lint.SeverityError, Message: "test"},
	}
	var buf bytes.Buffer
	Write(&buf, findings, FormatSARIF, "workflow.yaml")

	var report sarifReport
	json.Unmarshal(buf.Bytes(), &report)

	if len(report.Runs[0].Results[0].Locations) != 1 {
		t.Fatal("expected 1 location")
	}
	uri := report.Runs[0].Results[0].Locations[0].PhysicalLocation.ArtifactLocation.URI
	if uri != "workflow.yaml" {
		t.Errorf("expected URI 'workflow.yaml', got %q", uri)
	}
}

func TestWriteSARIF_EmptyFilePath(t *testing.T) {
	findings := []lint.Finding{
		{RuleID: "R1", Severity: lint.SeverityError, Message: "test"},
	}
	var buf bytes.Buffer
	Write(&buf, findings, FormatSARIF, "")

	var report sarifReport
	json.Unmarshal(buf.Bytes(), &report)

	if len(report.Runs[0].Results[0].Locations) != 0 {
		t.Error("expected no locations for empty file path")
	}
}

func TestWrite_UnsupportedFormat(t *testing.T) {
	var buf bytes.Buffer
	err := Write(&buf, nil, Format("xml"), "test.yaml")
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}
}

func TestWriteSARIF_DuplicateRules(t *testing.T) {
	findings := []lint.Finding{
		{RuleID: "DAG001", Severity: lint.SeverityError, Message: "cycle1"},
		{RuleID: "DAG001", Severity: lint.SeverityError, Message: "cycle2"},
	}
	var buf bytes.Buffer
	Write(&buf, findings, FormatSARIF, "test.yaml")

	var report sarifReport
	json.Unmarshal(buf.Bytes(), &report)

	// Should only have 1 rule definition even with 2 findings
	if len(report.Runs[0].Tool.Driver.Rules) != 1 {
		t.Errorf("expected 1 rule (dedup), got %d", len(report.Runs[0].Tool.Driver.Rules))
	}
	if len(report.Runs[0].Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(report.Runs[0].Results))
	}
}
