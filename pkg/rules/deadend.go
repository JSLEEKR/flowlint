package rules

import (
	"fmt"

	"github.com/JSLEEKR/flowlint/pkg/dag"
	"github.com/JSLEEKR/flowlint/pkg/lint"
)

// DeadEndDetection checks for steps that produce output but have no downstream consumers.
type DeadEndDetection struct{}

func (r *DeadEndDetection) ID() string          { return "DAG003" }
func (r *DeadEndDetection) Description() string { return "Detects dead-end steps with unconsumed output" }
func (r *DeadEndDetection) DefaultSeverity() lint.Severity {
	return lint.SeverityWarning
}

func (r *DeadEndDetection) Check(g *dag.Graph) []lint.Finding {
	deadEnds := g.FindDeadEnds()
	if len(deadEnds) == 0 {
		return nil
	}

	var findings []lint.Finding
	for _, id := range deadEnds {
		findings = append(findings, lint.Finding{
			RuleID:   r.ID(),
			Severity: r.DefaultSeverity(),
			Message:  fmt.Sprintf("step %q produces output but has no downstream consumers", id),
			StepID:   id,
		})
	}
	return findings
}
