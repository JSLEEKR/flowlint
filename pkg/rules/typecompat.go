package rules

import (
	"fmt"
	"sort"

	"github.com/JSLEEKR/flowlint/pkg/dag"
	"github.com/JSLEEKR/flowlint/pkg/lint"
)

// TypeCompatibility checks that output types match downstream input expectations.
type TypeCompatibility struct{}

func (r *TypeCompatibility) ID() string          { return "TYPE001" }
func (r *TypeCompatibility) Description() string { return "Validates output-to-input type compatibility" }
func (r *TypeCompatibility) DefaultSeverity() lint.Severity {
	return lint.SeverityError
}

// compatibleFormats defines which format conversions are implicitly allowed.
var compatibleFormats = map[string]map[string]bool{
	"string": {"string": true},
	"number": {"number": true, "string": true},
	"boolean": {"boolean": true, "string": true},
	"json":   {"json": true, "object": true, "string": true},
	"object": {"object": true, "json": true, "string": true},
	"array":  {"array": true, "json": true, "string": true},
}

// isCompatible checks if source format can flow into target format.
func isCompatible(source, target string) bool {
	if source == target {
		return true
	}
	if source == "" || target == "" {
		// If either is unspecified, assume compatible
		return true
	}
	targets, ok := compatibleFormats[source]
	if !ok {
		return false
	}
	return targets[target]
}

func (r *TypeCompatibility) Check(g *dag.Graph) []lint.Finding {
	var findings []lint.Finding

	// Sort node IDs for deterministic output
	sortedIDs := make([]string, 0, len(g.InEdges))
	for id := range g.InEdges {
		sortedIDs = append(sortedIDs, id)
	}
	sort.Strings(sortedIDs)

	for _, id := range sortedIDs {
		deps := g.InEdges[id]
		node := g.Nodes[id]
		if node.Input == nil {
			continue
		}

		for _, depID := range deps {
			depNode := g.Nodes[depID]
			if depNode.Output == nil {
				findings = append(findings, lint.Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					Message: fmt.Sprintf(
						"step %q expects input (format=%q) but upstream step %q has no output defined",
						id, node.Input.Format, depID,
					),
					StepID: id,
				})
				continue
			}

			if !isCompatible(depNode.Output.Format, node.Input.Format) {
				findings = append(findings, lint.Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					Message: fmt.Sprintf(
						"type mismatch: step %q output format %q is incompatible with step %q input format %q",
						depID, depNode.Output.Format, id, node.Input.Format,
					),
					StepID: id,
				})
			}
		}
	}

	return findings
}
