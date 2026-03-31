package rules

import (
	"fmt"
	"sort"

	"github.com/JSLEEKR/flowlint/pkg/dag"
	"github.com/JSLEEKR/flowlint/pkg/lint"
)

// ErrorHandlerCoverage checks that steps have error handling configured.
type ErrorHandlerCoverage struct{}

func (r *ErrorHandlerCoverage) ID() string { return "ERR001" }
func (r *ErrorHandlerCoverage) Description() string {
	return "Flags steps without error handler configuration"
}
func (r *ErrorHandlerCoverage) DefaultSeverity() lint.Severity {
	return lint.SeverityWarning
}

func (r *ErrorHandlerCoverage) Check(g *dag.Graph) []lint.Finding {
	var findings []lint.Finding

	ids := sortedNodeIDs(g)
	for _, id := range ids {
		node := g.Nodes[id]
		if node.ErrorHandler == nil {
			findings = append(findings, lint.Finding{
				RuleID:   r.ID(),
				Severity: r.DefaultSeverity(),
				Message:  fmt.Sprintf("step %q has no error handler configured", id),
				StepID:   id,
			})
		}
	}

	return findings
}

// sortedNodeIDs returns node IDs in sorted order for deterministic output.
func sortedNodeIDs(g *dag.Graph) []string {
	ids := make([]string, 0, len(g.Nodes))
	for id := range g.Nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// TimeoutConfig checks that steps have timeout configured.
type TimeoutConfig struct{}

func (r *TimeoutConfig) ID() string          { return "ERR002" }
func (r *TimeoutConfig) Description() string { return "Flags steps without timeout configuration" }
func (r *TimeoutConfig) DefaultSeverity() lint.Severity {
	return lint.SeverityWarning
}

func (r *TimeoutConfig) Check(g *dag.Graph) []lint.Finding {
	var findings []lint.Finding

	for _, id := range sortedNodeIDs(g) {
		node := g.Nodes[id]
		hasTimeout := node.Timeout != ""
		if node.ErrorHandler != nil && node.ErrorHandler.Timeout != "" {
			hasTimeout = true
		}
		if !hasTimeout {
			findings = append(findings, lint.Finding{
				RuleID:   r.ID(),
				Severity: r.DefaultSeverity(),
				Message:  fmt.Sprintf("step %q has no timeout configured", id),
				StepID:   id,
			})
		}
	}

	return findings
}

// RetryConfig checks for steps with retry without proper configuration.
type RetryConfig struct{}

func (r *RetryConfig) ID() string          { return "ERR003" }
func (r *RetryConfig) Description() string { return "Validates retry configuration consistency" }
func (r *RetryConfig) DefaultSeverity() lint.Severity {
	return lint.SeverityWarning
}

func (r *RetryConfig) Check(g *dag.Graph) []lint.Finding {
	var findings []lint.Finding

	for _, id := range sortedNodeIDs(g) {
		node := g.Nodes[id]
		// Check if retry is set on step but no error handler
		if node.Retry > 0 && node.ErrorHandler == nil {
			findings = append(findings, lint.Finding{
				RuleID:   r.ID(),
				Severity: r.DefaultSeverity(),
				Message:  fmt.Sprintf("step %q has retry=%d but no error handler to define retry behavior", id, node.Retry),
				StepID:   id,
			})
		}

		// Check if error handler has retry strategy but no max_retry
		if node.ErrorHandler != nil && node.ErrorHandler.Strategy == "retry" && node.ErrorHandler.MaxRetry == 0 && node.Retry == 0 {
			findings = append(findings, lint.Finding{
				RuleID:   r.ID(),
				Severity: r.DefaultSeverity(),
				Message:  fmt.Sprintf("step %q has retry strategy but no max_retry count specified", id),
				StepID:   id,
			})
		}

		// Check if error handler has fallback strategy but no fallback step
		if node.ErrorHandler != nil && node.ErrorHandler.Strategy == "fallback" && node.ErrorHandler.Fallback == "" {
			findings = append(findings, lint.Finding{
				RuleID:   r.ID(),
				Severity: lint.SeverityError,
				Message:  fmt.Sprintf("step %q has fallback strategy but no fallback step specified", id),
				StepID:   id,
			})
		}

		// Check if fallback references a valid step
		if node.ErrorHandler != nil && node.ErrorHandler.Fallback != "" {
			if _, exists := g.Nodes[node.ErrorHandler.Fallback]; !exists {
				findings = append(findings, lint.Finding{
					RuleID:   r.ID(),
					Severity: lint.SeverityError,
					Message:  fmt.Sprintf("step %q references fallback step %q which does not exist", id, node.ErrorHandler.Fallback),
					StepID:   id,
				})
			}
		}
	}

	return findings
}
