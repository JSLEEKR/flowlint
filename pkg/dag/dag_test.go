package dag

import (
	"testing"
)

func TestBuildGraph_Simple(t *testing.T) {
	w := &Workflow{
		Name: "test",
		Steps: []Step{
			{ID: "a"},
			{ID: "b", DependsOn: []string{"a"}},
			{ID: "c", DependsOn: []string{"b"}},
		},
	}

	g, err := BuildGraph(w)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(g.Nodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(g.Nodes))
	}
	if len(g.Roots) != 1 || g.Roots[0] != "a" {
		t.Errorf("expected root [a], got %v", g.Roots)
	}
	if len(g.Leaves) != 1 || g.Leaves[0] != "c" {
		t.Errorf("expected leaf [c], got %v", g.Leaves)
	}
}

func TestBuildGraph_EmptyID(t *testing.T) {
	w := &Workflow{Steps: []Step{{ID: ""}}}
	_, err := BuildGraph(w)
	if err == nil {
		t.Fatal("expected error for empty ID")
	}
}

func TestBuildGraph_DuplicateID(t *testing.T) {
	w := &Workflow{Steps: []Step{{ID: "a"}, {ID: "a"}}}
	_, err := BuildGraph(w)
	if err == nil {
		t.Fatal("expected error for duplicate ID")
	}
}

func TestBuildGraph_UnknownDependency(t *testing.T) {
	w := &Workflow{Steps: []Step{{ID: "a", DependsOn: []string{"nonexistent"}}}}
	_, err := BuildGraph(w)
	if err == nil {
		t.Fatal("expected error for unknown dependency")
	}
}

func TestBuildGraph_MultipleRoots(t *testing.T) {
	w := &Workflow{
		Steps: []Step{
			{ID: "a"},
			{ID: "b"},
			{ID: "c", DependsOn: []string{"a", "b"}},
		},
	}
	g, err := BuildGraph(w)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(g.Roots) != 2 {
		t.Errorf("expected 2 roots, got %d: %v", len(g.Roots), g.Roots)
	}
}

func TestDetectCycles_NoCycle(t *testing.T) {
	w := &Workflow{
		Steps: []Step{
			{ID: "a"},
			{ID: "b", DependsOn: []string{"a"}},
			{ID: "c", DependsOn: []string{"b"}},
		},
	}
	g, _ := BuildGraph(w)
	cycles := g.DetectCycles()
	if len(cycles) != 0 {
		t.Errorf("expected no cycles, got %v", cycles)
	}
}

func TestDetectCycles_SimpleCycle(t *testing.T) {
	// Create cycle manually since BuildGraph validates deps
	g := &Graph{
		Nodes: map[string]*Step{
			"a": {ID: "a"},
			"b": {ID: "b"},
		},
		Edges: map[string][]string{
			"a": {"b"},
			"b": {"a"},
		},
		InEdges: map[string][]string{
			"a": {"b"},
			"b": {"a"},
		},
	}
	cycles := g.DetectCycles()
	if len(cycles) != 2 {
		t.Errorf("expected 2 nodes in cycle, got %v", cycles)
	}
}

func TestDetectCycles_ThreeNodeCycle(t *testing.T) {
	g := &Graph{
		Nodes: map[string]*Step{
			"a": {ID: "a"},
			"b": {ID: "b"},
			"c": {ID: "c"},
		},
		Edges: map[string][]string{
			"a": {"b"},
			"b": {"c"},
			"c": {"a"},
		},
		InEdges: map[string][]string{
			"a": {"c"},
			"b": {"a"},
			"c": {"b"},
		},
	}
	cycles := g.DetectCycles()
	if len(cycles) != 3 {
		t.Errorf("expected 3 nodes in cycle, got %v", cycles)
	}
}

func TestDetectCycles_PartialCycle(t *testing.T) {
	// a -> b -> c -> b (cycle), d depends on a (not in cycle)
	g := &Graph{
		Nodes: map[string]*Step{
			"a": {ID: "a"},
			"b": {ID: "b"},
			"c": {ID: "c"},
			"d": {ID: "d"},
		},
		Edges: map[string][]string{
			"a": {"b", "d"},
			"b": {"c"},
			"c": {"b"},
			"d": {},
		},
		InEdges: map[string][]string{
			"a": {},
			"b": {"a", "c"},
			"c": {"b"},
			"d": {"a"},
		},
	}
	cycles := g.DetectCycles()
	if len(cycles) != 2 {
		t.Errorf("expected 2 nodes in cycle (b, c), got %v", cycles)
	}
}

func TestFindUnreachableNodes_AllReachable(t *testing.T) {
	w := &Workflow{
		Steps: []Step{
			{ID: "a"},
			{ID: "b", DependsOn: []string{"a"}},
		},
	}
	g, _ := BuildGraph(w)
	unreachable := g.FindUnreachableNodes()
	if len(unreachable) != 0 {
		t.Errorf("expected no unreachable nodes, got %v", unreachable)
	}
}

func TestFindUnreachableNodes_WithUnreachable(t *testing.T) {
	// x -> y forms a subgraph not connected to root a
	g := &Graph{
		Nodes: map[string]*Step{
			"a": {ID: "a"},
			"x": {ID: "x"},
			"y": {ID: "y"},
		},
		Edges: map[string][]string{
			"a": {},
			"x": {"y"},
			"y": {},
		},
		InEdges: map[string][]string{
			"a": {},
			"x": {},
			"y": {"x"},
		},
		Roots: []string{"a", "x"},
	}
	// Both a and x are roots, so all are reachable
	unreachable := g.FindUnreachableNodes()
	if len(unreachable) != 0 {
		t.Errorf("expected no unreachable, got %v", unreachable)
	}
}

func TestFindUnreachableNodes_NoRoots(t *testing.T) {
	g := &Graph{
		Nodes: map[string]*Step{
			"a": {ID: "a"},
			"b": {ID: "b"},
		},
		Edges: map[string][]string{
			"a": {"b"},
			"b": {"a"},
		},
		InEdges: map[string][]string{
			"a": {"b"},
			"b": {"a"},
		},
		Roots: nil,
	}
	unreachable := g.FindUnreachableNodes()
	if len(unreachable) != 2 {
		t.Errorf("expected 2 unreachable, got %v", unreachable)
	}
}

func TestFindDeadEnds_NoDeadEnds(t *testing.T) {
	w := &Workflow{
		Steps: []Step{
			{ID: "a"},
			{ID: "b", DependsOn: []string{"a"}},
		},
	}
	g, _ := BuildGraph(w)
	deadEnds := g.FindDeadEnds()
	if len(deadEnds) != 0 {
		t.Errorf("expected no dead ends, got %v", deadEnds)
	}
}

func TestFindDeadEnds_WithDeadEnd(t *testing.T) {
	w := &Workflow{
		Steps: []Step{
			{ID: "a"},
			{ID: "b", DependsOn: []string{"a"}, Output: &TypeInfo{Name: "result", Format: "json"}},
		},
	}
	g, _ := BuildGraph(w)
	deadEnds := g.FindDeadEnds()
	if len(deadEnds) != 1 || deadEnds[0] != "b" {
		t.Errorf("expected dead end [b], got %v", deadEnds)
	}
}

func TestFindOrphanSteps_NoOrphans(t *testing.T) {
	w := &Workflow{
		Steps: []Step{
			{ID: "a"},
			{ID: "b", DependsOn: []string{"a"}},
		},
	}
	g, _ := BuildGraph(w)
	orphans := g.FindOrphanSteps()
	if len(orphans) != 0 {
		t.Errorf("expected no orphans, got %v", orphans)
	}
}

func TestFindOrphanSteps_WithOrphans(t *testing.T) {
	w := &Workflow{
		Steps: []Step{
			{ID: "a"},
			{ID: "b", DependsOn: []string{"a"}},
			{ID: "orphan"},
		},
	}
	g, _ := BuildGraph(w)
	orphans := g.FindOrphanSteps()
	// 'a' has dependents so not orphan, 'orphan' has neither
	if len(orphans) != 1 || orphans[0] != "orphan" {
		t.Errorf("expected orphan [orphan], got %v", orphans)
	}
}

func TestFindOrphanSteps_SingleNode(t *testing.T) {
	w := &Workflow{Steps: []Step{{ID: "only"}}}
	g, _ := BuildGraph(w)
	orphans := g.FindOrphanSteps()
	if len(orphans) != 0 {
		t.Errorf("single-node workflow should not report orphans, got %v", orphans)
	}
}

func TestFindOrphanSteps_AllDisconnected(t *testing.T) {
	w := &Workflow{
		Steps: []Step{
			{ID: "a"},
			{ID: "b"},
			{ID: "c"},
		},
	}
	g, _ := BuildGraph(w)
	orphans := g.FindOrphanSteps()
	if len(orphans) != 3 {
		t.Errorf("expected 3 orphans, got %v", orphans)
	}
}

func TestTopologicalSort_Simple(t *testing.T) {
	w := &Workflow{
		Steps: []Step{
			{ID: "a"},
			{ID: "b", DependsOn: []string{"a"}},
			{ID: "c", DependsOn: []string{"b"}},
		},
	}
	g, _ := BuildGraph(w)
	order, err := g.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) != 3 {
		t.Errorf("expected 3 items, got %d", len(order))
	}
	// a must come before b, b before c
	pos := make(map[string]int)
	for i, id := range order {
		pos[id] = i
	}
	if pos["a"] > pos["b"] || pos["b"] > pos["c"] {
		t.Errorf("invalid order: %v", order)
	}
}

func TestTopologicalSort_WithCycle(t *testing.T) {
	g := &Graph{
		Nodes: map[string]*Step{
			"a": {ID: "a"},
			"b": {ID: "b"},
		},
		Edges: map[string][]string{
			"a": {"b"},
			"b": {"a"},
		},
		InEdges: map[string][]string{
			"a": {"b"},
			"b": {"a"},
		},
	}
	_, err := g.TopologicalSort()
	if err == nil {
		t.Fatal("expected error for cycle")
	}
}

func TestTopologicalSort_Diamond(t *testing.T) {
	w := &Workflow{
		Steps: []Step{
			{ID: "a"},
			{ID: "b", DependsOn: []string{"a"}},
			{ID: "c", DependsOn: []string{"a"}},
			{ID: "d", DependsOn: []string{"b", "c"}},
		},
	}
	g, _ := BuildGraph(w)
	order, err := g.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pos := make(map[string]int)
	for i, id := range order {
		pos[id] = i
	}
	if pos["a"] > pos["b"] || pos["a"] > pos["c"] || pos["b"] > pos["d"] || pos["c"] > pos["d"] {
		t.Errorf("invalid diamond order: %v", order)
	}
}

func TestTopologicalSort_DeterministicOrder(t *testing.T) {
	// Regression test: nodes freed at different processing stages must
	// still appear in lexicographic order among valid alternatives.
	// x -> a -> b, x -> z
	// Both [x,a,b,z] and [x,a,z,b] are valid topological orders,
	// but deterministic sort should produce [x,a,b,z].
	w := &Workflow{
		Steps: []Step{
			{ID: "x"},
			{ID: "a", DependsOn: []string{"x"}},
			{ID: "z", DependsOn: []string{"x"}},
			{ID: "b", DependsOn: []string{"a"}},
		},
	}
	g, _ := BuildGraph(w)
	order, err := g.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{"x", "a", "b", "z"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d items, got %d: %v", len(expected), len(order), order)
	}
	for i, id := range expected {
		if order[i] != id {
			t.Errorf("position %d: expected %q, got %q (full order: %v)", i, id, order[i], order)
		}
	}
}

func TestBuildGraph_ComplexWorkflow(t *testing.T) {
	w := &Workflow{
		Name:    "complex",
		Version: "1.0",
		Steps: []Step{
			{ID: "fetch"},
			{ID: "parse", DependsOn: []string{"fetch"}},
			{ID: "validate", DependsOn: []string{"parse"}},
			{ID: "transform", DependsOn: []string{"validate"}},
			{ID: "notify", DependsOn: []string{"validate"}},
			{ID: "store", DependsOn: []string{"transform", "notify"}},
		},
	}
	g, err := BuildGraph(w)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(g.Roots) != 1 || g.Roots[0] != "fetch" {
		t.Errorf("expected root [fetch], got %v", g.Roots)
	}
	if len(g.Leaves) != 1 || g.Leaves[0] != "store" {
		t.Errorf("expected leaf [store], got %v", g.Leaves)
	}
}
