// Package lint defines the linting engine that runs rules against a workflow DAG.
package lint

import (
	"fmt"
	"sort"

	"github.com/JSLEEKR/flowlint/pkg/dag"
)

// Severity represents the severity level of a lint finding.
type Severity int

const (
	SeverityInfo    Severity = 0
	SeverityWarning Severity = 1
	SeverityError   Severity = 2
)

// String returns the string representation of a severity.
func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	default:
		return "unknown"
	}
}

// Finding represents a single lint result.
type Finding struct {
	RuleID   string   `json:"rule_id"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	StepID   string   `json:"step_id,omitempty"`
	StepIDs  []string `json:"step_ids,omitempty"`
}

// Rule is the interface that all lint rules must implement.
type Rule interface {
	// ID returns the unique identifier for this rule.
	ID() string
	// Description returns a human-readable description of the rule.
	Description() string
	// Severity returns the default severity of findings from this rule.
	DefaultSeverity() Severity
	// Check runs the rule against the given graph and returns findings.
	Check(g *dag.Graph) []Finding
}

// Engine is the lint engine that manages and executes rules.
type Engine struct {
	rules    []Rule
	disabled map[string]bool
}

// NewEngine creates a new lint engine.
func NewEngine() *Engine {
	return &Engine{
		disabled: make(map[string]bool),
	}
}

// AddRule registers a rule with the engine.
func (e *Engine) AddRule(r Rule) {
	e.rules = append(e.rules, r)
}

// DisableRule disables a rule by ID.
func (e *Engine) DisableRule(id string) {
	e.disabled[id] = true
}

// EnableRule enables a previously disabled rule.
func (e *Engine) EnableRule(id string) {
	delete(e.disabled, id)
}

// Rules returns the list of registered rules.
func (e *Engine) Rules() []Rule {
	return e.rules
}

// Run executes all enabled rules against the given graph and returns findings.
func (e *Engine) Run(g *dag.Graph) []Finding {
	var findings []Finding

	for _, r := range e.rules {
		if e.disabled[r.ID()] {
			continue
		}
		results := r.Check(g)
		findings = append(findings, results...)
	}

	// Sort findings by severity (errors first), then by rule ID
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Severity != findings[j].Severity {
			return findings[i].Severity > findings[j].Severity
		}
		return findings[i].RuleID < findings[j].RuleID
	})

	return findings
}

// HasErrors returns true if any finding has error severity.
func HasErrors(findings []Finding) bool {
	for _, f := range findings {
		if f.Severity == SeverityError {
			return true
		}
	}
	return false
}

// HasWarnings returns true if any finding has warning severity or above.
func HasWarnings(findings []Finding) bool {
	for _, f := range findings {
		if f.Severity >= SeverityWarning {
			return true
		}
	}
	return false
}

// Summary returns a summary string of the findings.
func Summary(findings []Finding) string {
	errors := 0
	warnings := 0
	infos := 0
	for _, f := range findings {
		switch f.Severity {
		case SeverityError:
			errors++
		case SeverityWarning:
			warnings++
		case SeverityInfo:
			infos++
		}
	}
	return fmt.Sprintf("%d error(s), %d warning(s), %d info(s)", errors, warnings, infos)
}
