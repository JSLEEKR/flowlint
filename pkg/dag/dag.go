// Package dag provides DAG data structures and graph analysis algorithms
// for workflow validation.
package dag

import (
	"fmt"
	"sort"
	"strings"
)

// StepType represents the type of a workflow step.
type StepType string

const (
	StepTypeTask     StepType = "task"
	StepTypeDecision StepType = "decision"
	StepTypeParallel StepType = "parallel"
	StepTypeWait     StepType = "wait"
)

// TypeInfo describes the input/output type contract for a step.
type TypeInfo struct {
	Name   string `json:"name" yaml:"name"`
	Format string `json:"format,omitempty" yaml:"format,omitempty"` // e.g., "json", "string", "number", "boolean", "object", "array"
}

// ErrorHandler describes error handling configuration for a step.
type ErrorHandler struct {
	Strategy string `json:"strategy,omitempty" yaml:"strategy,omitempty"` // "retry", "fallback", "ignore", "abort"
	MaxRetry int    `json:"max_retry,omitempty" yaml:"max_retry,omitempty"`
	Timeout  string `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Fallback string `json:"fallback,omitempty" yaml:"fallback,omitempty"`
}

// Step represents a single step/node in a workflow DAG.
type Step struct {
	ID           string        `json:"id" yaml:"id"`
	Name         string        `json:"name,omitempty" yaml:"name,omitempty"`
	Type         StepType      `json:"type,omitempty" yaml:"type,omitempty"`
	DependsOn    []string      `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
	Input        *TypeInfo     `json:"input,omitempty" yaml:"input,omitempty"`
	Output       *TypeInfo     `json:"output,omitempty" yaml:"output,omitempty"`
	ErrorHandler *ErrorHandler `json:"error_handler,omitempty" yaml:"error_handler,omitempty"`
	Timeout      string        `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Retry        int           `json:"retry,omitempty" yaml:"retry,omitempty"`
}

// Workflow represents a parsed workflow definition containing steps and metadata.
type Workflow struct {
	Name        string `json:"name,omitempty" yaml:"name,omitempty"`
	Version     string `json:"version,omitempty" yaml:"version,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Steps       []Step `json:"steps" yaml:"steps"`
}

// Graph is the internal DAG representation built from a Workflow.
type Graph struct {
	Nodes    map[string]*Step
	Edges    map[string][]string // adjacency list: step -> its dependents
	InEdges  map[string][]string // reverse adjacency: step -> its dependencies
	Roots    []string            // nodes with no dependencies
	Leaves   []string            // nodes with no dependents
	workflow *Workflow
}

// BuildGraph constructs a Graph from a Workflow definition.
func BuildGraph(w *Workflow) (*Graph, error) {
	g := &Graph{
		Nodes:    make(map[string]*Step),
		Edges:    make(map[string][]string),
		InEdges:  make(map[string][]string),
		workflow: w,
	}

	// Index all steps
	for i := range w.Steps {
		step := &w.Steps[i]
		// Trim whitespace from step ID; reject if empty after trim
		step.ID = strings.TrimSpace(step.ID)
		if step.ID == "" {
			return nil, fmt.Errorf("step at index %d has empty ID", i)
		}
		if _, exists := g.Nodes[step.ID]; exists {
			return nil, fmt.Errorf("duplicate step ID: %q", step.ID)
		}
		g.Nodes[step.ID] = step
		g.Edges[step.ID] = nil
		g.InEdges[step.ID] = nil
	}

	// Build edges from depends_on (deduplicate entries)
	for _, step := range w.Steps {
		seen := make(map[string]bool)
		for _, dep := range step.DependsOn {
			if seen[dep] {
				continue
			}
			seen[dep] = true
			if _, exists := g.Nodes[dep]; !exists {
				return nil, fmt.Errorf("step %q depends on unknown step %q", step.ID, dep)
			}
			g.Edges[dep] = append(g.Edges[dep], step.ID)
			g.InEdges[step.ID] = append(g.InEdges[step.ID], dep)
		}
	}

	// Find roots and leaves
	for id := range g.Nodes {
		if len(g.InEdges[id]) == 0 {
			g.Roots = append(g.Roots, id)
		}
		if len(g.Edges[id]) == 0 {
			g.Leaves = append(g.Leaves, id)
		}
	}
	sort.Strings(g.Roots)
	sort.Strings(g.Leaves)

	return g, nil
}

// DetectCycles uses Kahn's algorithm to detect cycles in the graph.
// Returns the list of node IDs involved in cycles (empty if acyclic).
func (g *Graph) DetectCycles() []string {
	inDegree := make(map[string]int)
	for id := range g.Nodes {
		inDegree[id] = len(g.InEdges[id])
	}

	var queue []string
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	visited := 0
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		visited++
		for _, dep := range g.Edges[node] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	if visited == len(g.Nodes) {
		return nil
	}

	// Collect nodes still in cycles
	var cycleNodes []string
	for id, deg := range inDegree {
		if deg > 0 {
			cycleNodes = append(cycleNodes, id)
		}
	}
	sort.Strings(cycleNodes)
	return cycleNodes
}

// FindUnreachableNodes returns nodes not reachable from any root.
func (g *Graph) FindUnreachableNodes() []string {
	if len(g.Roots) == 0 {
		// If no roots, all nodes are unreachable (likely a cycle)
		var all []string
		for id := range g.Nodes {
			all = append(all, id)
		}
		sort.Strings(all)
		return all
	}

	reachable := make(map[string]bool)
	var dfs func(string)
	dfs = func(id string) {
		if reachable[id] {
			return
		}
		reachable[id] = true
		for _, dep := range g.Edges[id] {
			dfs(dep)
		}
	}

	for _, root := range g.Roots {
		dfs(root)
	}

	var unreachable []string
	for id := range g.Nodes {
		if !reachable[id] {
			unreachable = append(unreachable, id)
		}
	}
	sort.Strings(unreachable)
	return unreachable
}

// FindDeadEnds returns non-leaf nodes that have no outgoing edges
// but are not the final steps of the workflow. A dead end is a node
// that has incoming edges but no outgoing edges AND is not a designated
// terminal node. For simplicity, we define dead ends as nodes with
// dependents that themselves have no further dependents and no outputs.
// Actually, dead ends are simply nodes with no dependents that are not
// explicitly terminal — but since leaves are natural endpoints, we
// look for nodes that are expected to continue but don't.
// We define dead ends as leaf nodes that have outputs defined but
// no downstream consumers.
func (g *Graph) FindDeadEnds() []string {
	var deadEnds []string
	for _, id := range g.Leaves {
		node := g.Nodes[id]
		if node.Output != nil && (node.Output.Format != "" || node.Output.Name != "") {
			// This node produces output but nothing consumes it
			deadEnds = append(deadEnds, id)
		}
	}
	sort.Strings(deadEnds)
	return deadEnds
}

// FindOrphanSteps returns steps that have no dependencies AND no dependents
// in a multi-step workflow (isolated nodes).
func (g *Graph) FindOrphanSteps() []string {
	if len(g.Nodes) <= 1 {
		return nil
	}

	var orphans []string
	for id := range g.Nodes {
		if len(g.InEdges[id]) == 0 && len(g.Edges[id]) == 0 {
			orphans = append(orphans, id)
		}
	}

	// If ALL nodes are orphans (completely disconnected graph), report all
	// If only some are orphans, report those
	if len(orphans) == len(g.Nodes) {
		sort.Strings(orphans)
		return orphans
	}

	sort.Strings(orphans)
	return orphans
}

// TopologicalSort returns nodes in topological order. Returns error if cycles exist.
func (g *Graph) TopologicalSort() ([]string, error) {
	cycles := g.DetectCycles()
	if len(cycles) > 0 {
		return nil, fmt.Errorf("cannot topologically sort: cycle detected involving nodes: %v", cycles)
	}

	inDegree := make(map[string]int)
	for id := range g.Nodes {
		inDegree[id] = len(g.InEdges[id])
	}

	var queue []string
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}
	sort.Strings(queue)

	var result []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)

		for _, dep := range g.Edges[node] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
		// Re-sort queue to maintain deterministic lexicographic ordering.
		// Without this, nodes freed at different processing stages would appear
		// in insertion order rather than alphabetical order.
		sort.Strings(queue)
	}

	return result, nil
}
