// Package report provides output formatters for lint findings.
package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/JSLEEKR/flowlint/pkg/lint"
)

// Format represents an output format type.
type Format string

const (
	FormatText  Format = "text"
	FormatJSON  Format = "json"
	FormatSARIF Format = "sarif"
)

// ParseFormat converts a string to a Format, returning error if invalid.
func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(s) {
	case "text", "":
		return FormatText, nil
	case "json":
		return FormatJSON, nil
	case "sarif":
		return FormatSARIF, nil
	default:
		return "", fmt.Errorf("unsupported output format: %q (use text, json, or sarif)", s)
	}
}

// Write writes findings in the specified format to the writer.
func Write(w io.Writer, findings []lint.Finding, format Format, filePath string) error {
	switch format {
	case FormatText:
		return writeText(w, findings, filePath)
	case FormatJSON:
		return writeJSON(w, findings, filePath)
	case FormatSARIF:
		return writeSARIF(w, findings, filePath)
	default:
		return fmt.Errorf("unsupported format: %q", format)
	}
}

func writeText(w io.Writer, findings []lint.Finding, filePath string) error {
	if len(findings) == 0 {
		_, err := fmt.Fprintf(w, "%s: no issues found\n", filePath)
		return err
	}

	for _, f := range findings {
		icon := severityIcon(f.Severity)
		stepInfo := ""
		if f.StepID != "" {
			stepInfo = fmt.Sprintf(" [step: %s]", f.StepID)
		} else if len(f.StepIDs) > 0 {
			stepInfo = fmt.Sprintf(" [steps: %s]", strings.Join(f.StepIDs, ", "))
		}
		_, err := fmt.Fprintf(w, "%s %s %s:%s %s\n", icon, f.Severity, filePath, stepInfo, f.Message)
		if err != nil {
			return err
		}
	}
	return nil
}

func severityIcon(s lint.Severity) string {
	switch s {
	case lint.SeverityError:
		return "E"
	case lint.SeverityWarning:
		return "W"
	case lint.SeverityInfo:
		return "I"
	default:
		return "?"
	}
}

// JSONReport is the structure for JSON output.
type JSONReport struct {
	File     string         `json:"file"`
	Findings []lint.Finding `json:"findings"`
	Summary  string         `json:"summary"`
}

func writeJSON(w io.Writer, findings []lint.Finding, filePath string) error {
	if findings == nil {
		findings = []lint.Finding{}
	}
	report := JSONReport{
		File:     filePath,
		Findings: findings,
		Summary:  lint.Summary(findings),
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

// SARIF structures for Static Analysis Results Interchange Format.
type sarifReport struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version"`
	InformationURI string      `json:"informationUri"`
	Rules          []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID               string         `json:"id"`
	ShortDescription sarifMultitext `json:"shortDescription"`
}

type sarifMultitext struct {
	Text string `json:"text"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifMultitext  `json:"message"`
	Locations []sarifLocation `json:"locations,omitempty"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifact `json:"artifactLocation"`
}

type sarifArtifact struct {
	URI string `json:"uri"`
}

func writeSARIF(w io.Writer, findings []lint.Finding, filePath string) error {
	// Collect unique rules
	ruleMap := make(map[string]bool)
	sarifRules := make([]sarifRule, 0)
	for _, f := range findings {
		if !ruleMap[f.RuleID] {
			ruleMap[f.RuleID] = true
			sarifRules = append(sarifRules, sarifRule{
				ID:               f.RuleID,
				ShortDescription: sarifMultitext{Text: f.RuleID},
			})
		}
	}

	results := make([]sarifResult, 0)
	for _, f := range findings {
		result := sarifResult{
			RuleID:  f.RuleID,
			Level:   sarifLevel(f.Severity),
			Message: sarifMultitext{Text: f.Message},
		}
		if filePath != "" {
			result.Locations = []sarifLocation{
				{
					PhysicalLocation: sarifPhysicalLocation{
						ArtifactLocation: sarifArtifact{URI: filePath},
					},
				},
			}
		}
		results = append(results, result)
	}

	report := sarifReport{
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/main/sarif-2.1/schema/sarif-schema-2.1.0.json",
		Version: "2.1.0",
		Runs: []sarifRun{
			{
				Tool: sarifTool{
					Driver: sarifDriver{
						Name:           "flowlint",
						Version:        "1.0.0",
						InformationURI: "https://github.com/JSLEEKR/flowlint",
						Rules:          sarifRules,
					},
				},
				Results: results,
			},
		},
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

func sarifLevel(s lint.Severity) string {
	switch s {
	case lint.SeverityError:
		return "error"
	case lint.SeverityWarning:
		return "warning"
	case lint.SeverityInfo:
		return "note"
	default:
		return "none"
	}
}
