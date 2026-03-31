// Package rules implements individual lint rules for workflow validation.
package rules

import (
	"fmt"
	"strings"

	"github.com/JSLEEKR/flowlint/pkg/dag"
	"github.com/JSLEEKR/flowlint/pkg/lint"
)

// CycleDetection checks for cycles in the workflow DAG.
type CycleDetection struct{}

func (r *CycleDetection) ID() string          { return "DAG001" }
func (r *CycleDetection) Description() string { return "Detects cycles in workflow DAG" }
func (r *CycleDetection) DefaultSeverity() lint.Severity {
	return lint.SeverityError
}

func (r *CycleDetection) Check(g *dag.Graph) []lint.Finding {
	cycleNodes := g.DetectCycles()
	if len(cycleNodes) == 0 {
		return nil
	}

	return []lint.Finding{
		{
			RuleID:   r.ID(),
			Severity: r.DefaultSeverity(),
			Message:  fmt.Sprintf("cycle detected involving steps: %s", strings.Join(cycleNodes, ", ")),
			StepIDs:  cycleNodes,
		},
	}
}
