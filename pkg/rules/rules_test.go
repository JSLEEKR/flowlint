package rules

import (
	"testing"

	"github.com/JSLEEKR/flowlint/pkg/dag"
	"github.com/JSLEEKR/flowlint/pkg/lint"
)

func buildGraph(t *testing.T, w *dag.Workflow) *dag.Graph {
	t.Helper()
	g, err := dag.BuildGraph(w)
	if err != nil {
		t.Fatalf("failed to build graph: %v", err)
	}
	return g
}

// --- CycleDetection tests ---

func TestCycleDetection_NoCycle(t *testing.T) {
	g := buildGraph(t, &dag.Workflow{
		Steps: []dag.Step{
			{ID: "a"},
			{ID: "b", DependsOn: []string{"a"}},
		},
	})
	r := &CycleDetection{}
	findings := r.Check(g)
	if len(findings) != 0 {
		t.Errorf("expected no findings, got %d", len(findings))
	}
}

func TestCycleDetection_WithCycle(t *testing.T) {
	g := &dag.Graph{
		Nodes: map[string]*dag.Step{
			"a": {ID: "a"},
			"b": {ID: "b"},
		},
		Edges:   map[string][]string{"a": {"b"}, "b": {"a"}},
		InEdges: map[string][]string{"a": {"b"}, "b": {"a"}},
	}
	r := &CycleDetection{}
	findings := r.Check(g)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Severity != lint.SeverityError {
		t.Errorf("expected error severity, got %v", findings[0].Severity)
	}
	if findings[0].RuleID != "DAG001" {
		t.Errorf("expected rule DAG001, got %q", findings[0].RuleID)
	}
}

// --- UnreachableNodes tests ---

func TestUnreachableNodes_AllReachable(t *testing.T) {
	g := buildGraph(t, &dag.Workflow{
		Steps: []dag.Step{
			{ID: "a"},
			{ID: "b", DependsOn: []string{"a"}},
		},
	})
	r := &UnreachableNodes{}
	findings := r.Check(g)
	if len(findings) != 0 {
		t.Errorf("expected no findings, got %d", len(findings))
	}
}

func TestUnreachableNodes_WithUnreachable(t *testing.T) {
	// Manually create a graph with unreachable nodes
	g := &dag.Graph{
		Nodes: map[string]*dag.Step{
			"a": {ID: "a"},
			"b": {ID: "b"},
			"c": {ID: "c"},
		},
		Edges: map[string][]string{
			"a": {},
			"b": {"c"},
			"c": {},
		},
		InEdges: map[string][]string{
			"a": {},
			"b": {"c"}, // c -> b makes b unreachable if only a is root
			"c": {"b"},
		},
		Roots: []string{"a"},
	}
	r := &UnreachableNodes{}
	findings := r.Check(g)
	if len(findings) != 2 {
		t.Errorf("expected 2 findings, got %d", len(findings))
	}
}

// --- DeadEndDetection tests ---

func TestDeadEndDetection_NoDeadEnds(t *testing.T) {
	g := buildGraph(t, &dag.Workflow{
		Steps: []dag.Step{
			{ID: "a"},
			{ID: "b", DependsOn: []string{"a"}},
		},
	})
	r := &DeadEndDetection{}
	findings := r.Check(g)
	if len(findings) != 0 {
		t.Errorf("expected no findings, got %d", len(findings))
	}
}

func TestDeadEndDetection_WithDeadEnd(t *testing.T) {
	g := buildGraph(t, &dag.Workflow{
		Steps: []dag.Step{
			{ID: "a"},
			{ID: "b", DependsOn: []string{"a"}, Output: &dag.TypeInfo{Name: "out", Format: "json"}},
		},
	})
	r := &DeadEndDetection{}
	findings := r.Check(g)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].StepID != "b" {
		t.Errorf("expected step b, got %q", findings[0].StepID)
	}
}

// --- OrphanSteps tests ---

func TestOrphanSteps_NoOrphans(t *testing.T) {
	g := buildGraph(t, &dag.Workflow{
		Steps: []dag.Step{
			{ID: "a"},
			{ID: "b", DependsOn: []string{"a"}},
		},
	})
	r := &OrphanSteps{}
	findings := r.Check(g)
	if len(findings) != 0 {
		t.Errorf("expected no findings, got %d", len(findings))
	}
}

func TestOrphanSteps_WithOrphan(t *testing.T) {
	g := buildGraph(t, &dag.Workflow{
		Steps: []dag.Step{
			{ID: "a"},
			{ID: "b", DependsOn: []string{"a"}},
			{ID: "orphan"},
		},
	})
	r := &OrphanSteps{}
	findings := r.Check(g)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].StepID != "orphan" {
		t.Errorf("expected step orphan, got %q", findings[0].StepID)
	}
}

// --- TypeCompatibility tests ---

func TestTypeCompatibility_Compatible(t *testing.T) {
	g := buildGraph(t, &dag.Workflow{
		Steps: []dag.Step{
			{ID: "a", Output: &dag.TypeInfo{Name: "out", Format: "json"}},
			{ID: "b", DependsOn: []string{"a"}, Input: &dag.TypeInfo{Name: "in", Format: "json"}},
		},
	})
	r := &TypeCompatibility{}
	findings := r.Check(g)
	if len(findings) != 0 {
		t.Errorf("expected no findings, got %d", len(findings))
	}
}

func TestTypeCompatibility_Incompatible(t *testing.T) {
	g := buildGraph(t, &dag.Workflow{
		Steps: []dag.Step{
			{ID: "a", Output: &dag.TypeInfo{Name: "out", Format: "boolean"}},
			{ID: "b", DependsOn: []string{"a"}, Input: &dag.TypeInfo{Name: "in", Format: "array"}},
		},
	})
	r := &TypeCompatibility{}
	findings := r.Check(g)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Severity != lint.SeverityError {
		t.Errorf("expected error, got %v", findings[0].Severity)
	}
}

func TestTypeCompatibility_MissingOutput(t *testing.T) {
	g := buildGraph(t, &dag.Workflow{
		Steps: []dag.Step{
			{ID: "a"},
			{ID: "b", DependsOn: []string{"a"}, Input: &dag.TypeInfo{Name: "in", Format: "json"}},
		},
	})
	r := &TypeCompatibility{}
	findings := r.Check(g)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
}

func TestTypeCompatibility_NoInput(t *testing.T) {
	g := buildGraph(t, &dag.Workflow{
		Steps: []dag.Step{
			{ID: "a", Output: &dag.TypeInfo{Name: "out", Format: "json"}},
			{ID: "b", DependsOn: []string{"a"}},
		},
	})
	r := &TypeCompatibility{}
	findings := r.Check(g)
	if len(findings) != 0 {
		t.Errorf("expected no findings when no input specified, got %d", len(findings))
	}
}

func TestTypeCompatibility_EmptyFormat(t *testing.T) {
	g := buildGraph(t, &dag.Workflow{
		Steps: []dag.Step{
			{ID: "a", Output: &dag.TypeInfo{Name: "out", Format: ""}},
			{ID: "b", DependsOn: []string{"a"}, Input: &dag.TypeInfo{Name: "in", Format: "json"}},
		},
	})
	r := &TypeCompatibility{}
	findings := r.Check(g)
	if len(findings) != 0 {
		t.Errorf("expected no findings for empty format (assume compatible), got %d", len(findings))
	}
}

func TestTypeCompatibility_NumberToString(t *testing.T) {
	g := buildGraph(t, &dag.Workflow{
		Steps: []dag.Step{
			{ID: "a", Output: &dag.TypeInfo{Name: "out", Format: "number"}},
			{ID: "b", DependsOn: []string{"a"}, Input: &dag.TypeInfo{Name: "in", Format: "string"}},
		},
	})
	r := &TypeCompatibility{}
	findings := r.Check(g)
	if len(findings) != 0 {
		t.Errorf("number->string should be compatible, got %d findings", len(findings))
	}
}

func TestTypeCompatibility_JSONToObject(t *testing.T) {
	g := buildGraph(t, &dag.Workflow{
		Steps: []dag.Step{
			{ID: "a", Output: &dag.TypeInfo{Name: "out", Format: "json"}},
			{ID: "b", DependsOn: []string{"a"}, Input: &dag.TypeInfo{Name: "in", Format: "object"}},
		},
	})
	r := &TypeCompatibility{}
	findings := r.Check(g)
	if len(findings) != 0 {
		t.Errorf("json->object should be compatible, got %d findings", len(findings))
	}
}

func TestTypeCompatibility_ArrayToNumber(t *testing.T) {
	g := buildGraph(t, &dag.Workflow{
		Steps: []dag.Step{
			{ID: "a", Output: &dag.TypeInfo{Name: "out", Format: "array"}},
			{ID: "b", DependsOn: []string{"a"}, Input: &dag.TypeInfo{Name: "in", Format: "number"}},
		},
	})
	r := &TypeCompatibility{}
	findings := r.Check(g)
	if len(findings) != 1 {
		t.Errorf("array->number should be incompatible, got %d findings", len(findings))
	}
}

// --- ErrorHandlerCoverage tests ---

func TestErrorHandlerCoverage_AllCovered(t *testing.T) {
	g := buildGraph(t, &dag.Workflow{
		Steps: []dag.Step{
			{ID: "a", ErrorHandler: &dag.ErrorHandler{Strategy: "retry"}},
		},
	})
	r := &ErrorHandlerCoverage{}
	findings := r.Check(g)
	if len(findings) != 0 {
		t.Errorf("expected no findings, got %d", len(findings))
	}
}

func TestErrorHandlerCoverage_Missing(t *testing.T) {
	g := buildGraph(t, &dag.Workflow{
		Steps: []dag.Step{
			{ID: "a"},
			{ID: "b", DependsOn: []string{"a"}},
		},
	})
	r := &ErrorHandlerCoverage{}
	findings := r.Check(g)
	if len(findings) != 2 {
		t.Errorf("expected 2 findings, got %d", len(findings))
	}
}

// --- TimeoutConfig tests ---

func TestTimeoutConfig_StepLevel(t *testing.T) {
	g := buildGraph(t, &dag.Workflow{
		Steps: []dag.Step{
			{ID: "a", Timeout: "30s"},
		},
	})
	r := &TimeoutConfig{}
	findings := r.Check(g)
	if len(findings) != 0 {
		t.Errorf("expected no findings, got %d", len(findings))
	}
}

func TestTimeoutConfig_ErrorHandlerLevel(t *testing.T) {
	g := buildGraph(t, &dag.Workflow{
		Steps: []dag.Step{
			{ID: "a", ErrorHandler: &dag.ErrorHandler{Timeout: "10s"}},
		},
	})
	r := &TimeoutConfig{}
	findings := r.Check(g)
	if len(findings) != 0 {
		t.Errorf("expected no findings, got %d", len(findings))
	}
}

func TestTimeoutConfig_Missing(t *testing.T) {
	g := buildGraph(t, &dag.Workflow{
		Steps: []dag.Step{
			{ID: "a"},
		},
	})
	r := &TimeoutConfig{}
	findings := r.Check(g)
	if len(findings) != 1 {
		t.Errorf("expected 1 finding, got %d", len(findings))
	}
}

// --- RetryConfig tests ---

func TestRetryConfig_RetryWithoutHandler(t *testing.T) {
	g := buildGraph(t, &dag.Workflow{
		Steps: []dag.Step{
			{ID: "a", Retry: 3},
		},
	})
	r := &RetryConfig{}
	findings := r.Check(g)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].RuleID != "ERR003" {
		t.Errorf("expected ERR003, got %q", findings[0].RuleID)
	}
}

func TestRetryConfig_RetryStrategyNoCount(t *testing.T) {
	g := buildGraph(t, &dag.Workflow{
		Steps: []dag.Step{
			{ID: "a", ErrorHandler: &dag.ErrorHandler{Strategy: "retry"}},
		},
	})
	r := &RetryConfig{}
	findings := r.Check(g)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
}

func TestRetryConfig_FallbackNoStep(t *testing.T) {
	g := buildGraph(t, &dag.Workflow{
		Steps: []dag.Step{
			{ID: "a", ErrorHandler: &dag.ErrorHandler{Strategy: "fallback"}},
		},
	})
	r := &RetryConfig{}
	findings := r.Check(g)
	hasError := false
	for _, f := range findings {
		if f.Severity == lint.SeverityError {
			hasError = true
		}
	}
	if !hasError {
		t.Error("expected error severity for missing fallback step")
	}
}

func TestRetryConfig_FallbackInvalidStep(t *testing.T) {
	g := buildGraph(t, &dag.Workflow{
		Steps: []dag.Step{
			{ID: "a", ErrorHandler: &dag.ErrorHandler{Strategy: "fallback", Fallback: "nonexistent"}},
		},
	})
	r := &RetryConfig{}
	findings := r.Check(g)
	hasError := false
	for _, f := range findings {
		if f.Severity == lint.SeverityError {
			hasError = true
		}
	}
	if !hasError {
		t.Error("expected error for nonexistent fallback step")
	}
}

func TestRetryConfig_ValidFallback(t *testing.T) {
	g := buildGraph(t, &dag.Workflow{
		Steps: []dag.Step{
			{ID: "a", ErrorHandler: &dag.ErrorHandler{Strategy: "fallback", Fallback: "b"}},
			{ID: "b"},
		},
	})
	r := &RetryConfig{}
	findings := r.Check(g)
	// Should have no error-severity findings for the fallback
	for _, f := range findings {
		if f.Severity == lint.SeverityError {
			t.Errorf("unexpected error: %s", f.Message)
		}
	}
}

func TestRetryConfig_ValidRetry(t *testing.T) {
	g := buildGraph(t, &dag.Workflow{
		Steps: []dag.Step{
			{ID: "a", Retry: 3, ErrorHandler: &dag.ErrorHandler{Strategy: "retry", MaxRetry: 3}},
		},
	})
	r := &RetryConfig{}
	findings := r.Check(g)
	if len(findings) != 0 {
		t.Errorf("expected no findings for valid retry config, got %d", len(findings))
	}
}

// --- Registry tests ---

func TestAllRules(t *testing.T) {
	allRules := AllRules()
	if len(allRules) != 8 {
		t.Errorf("expected 8 rules, got %d", len(allRules))
	}

	// Check all have unique IDs
	seen := make(map[string]bool)
	for _, r := range allRules {
		if seen[r.ID()] {
			t.Errorf("duplicate rule ID: %s", r.ID())
		}
		seen[r.ID()] = true
	}
}

func TestDefaultEngine(t *testing.T) {
	engine := DefaultEngine()
	if len(engine.Rules()) != 8 {
		t.Errorf("expected 8 rules in default engine, got %d", len(engine.Rules()))
	}
}

func TestDefaultEngine_RunsAllRules(t *testing.T) {
	engine := DefaultEngine()
	g := buildGraph(t, &dag.Workflow{
		Steps: []dag.Step{
			{ID: "a"},
			{ID: "b", DependsOn: []string{"a"}},
		},
	})
	findings := engine.Run(g)
	// Should have findings from error handler and timeout rules
	if len(findings) == 0 {
		t.Error("expected some findings from default rules")
	}
}
