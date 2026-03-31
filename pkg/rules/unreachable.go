package rules

import (
	"fmt"

	"github.com/JSLEEKR/flowlint/pkg/dag"
	"github.com/JSLEEKR/flowlint/pkg/lint"
)

// UnreachableNodes checks for nodes not reachable from any root.
type UnreachableNodes struct{}

func (r *UnreachableNodes) ID() string          { return "DAG002" }
func (r *UnreachableNodes) Description() string { return "Detects unreachable nodes in workflow" }
func (r *UnreachableNodes) DefaultSeverity() lint.Severity {
	return lint.SeverityError
}

func (r *UnreachableNodes) Check(g *dag.Graph) []lint.Finding {
	unreachable := g.FindUnreachableNodes()
	if len(unreachable) == 0 {
		return nil
	}

	var findings []lint.Finding
	for _, id := range unreachable {
		findings = append(findings, lint.Finding{
			RuleID:   r.ID(),
			Severity: r.DefaultSeverity(),
			Message:  fmt.Sprintf("step %q is not reachable from any root node", id),
			StepID:   id,
		})
	}
	return findings
}
