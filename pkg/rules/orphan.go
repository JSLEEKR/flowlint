package rules

import (
	"fmt"

	"github.com/JSLEEKR/flowlint/pkg/dag"
	"github.com/JSLEEKR/flowlint/pkg/lint"
)

// OrphanSteps checks for isolated steps with no connections to other steps.
type OrphanSteps struct{}

func (r *OrphanSteps) ID() string          { return "DAG004" }
func (r *OrphanSteps) Description() string { return "Detects orphan steps with no connections" }
func (r *OrphanSteps) DefaultSeverity() lint.Severity {
	return lint.SeverityWarning
}

func (r *OrphanSteps) Check(g *dag.Graph) []lint.Finding {
	orphans := g.FindOrphanSteps()
	if len(orphans) == 0 {
		return nil
	}

	var findings []lint.Finding
	for _, id := range orphans {
		findings = append(findings, lint.Finding{
			RuleID:   r.ID(),
			Severity: r.DefaultSeverity(),
			Message:  fmt.Sprintf("step %q is isolated (no dependencies and no dependents)", id),
			StepID:   id,
		})
	}
	return findings
}
