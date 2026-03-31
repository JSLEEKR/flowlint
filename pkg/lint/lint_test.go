package lint

import (
	"testing"

	"github.com/JSLEEKR/flowlint/pkg/dag"
)

// mockRule is a test rule that returns configured findings.
type mockRule struct {
	id       string
	desc     string
	severity Severity
	findings []Finding
}

func (r *mockRule) ID() string               { return r.id }
func (r *mockRule) Description() string       { return r.desc }
func (r *mockRule) DefaultSeverity() Severity { return r.severity }
func (r *mockRule) Check(_ *dag.Graph) []Finding {
	return r.findings
}

func TestSeverity_String(t *testing.T) {
	tests := []struct {
		s    Severity
		want string
	}{
		{SeverityInfo, "info"},
		{SeverityWarning, "warning"},
		{SeverityError, "error"},
		{Severity(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.s.String(); got != tt.want {
			t.Errorf("Severity(%d).String() = %q, want %q", tt.s, got, tt.want)
		}
	}
}

func TestEngine_AddAndRun(t *testing.T) {
	engine := NewEngine()
	engine.AddRule(&mockRule{
		id:       "TEST001",
		severity: SeverityError,
		findings: []Finding{{RuleID: "TEST001", Severity: SeverityError, Message: "test error"}},
	})

	g := &dag.Graph{Nodes: map[string]*dag.Step{}}
	findings := engine.Run(g)
	if len(findings) != 1 {
		t.Errorf("expected 1 finding, got %d", len(findings))
	}
}

func TestEngine_DisableRule(t *testing.T) {
	engine := NewEngine()
	engine.AddRule(&mockRule{
		id:       "TEST001",
		findings: []Finding{{RuleID: "TEST001", Severity: SeverityError, Message: "test"}},
	})
	engine.DisableRule("TEST001")

	g := &dag.Graph{Nodes: map[string]*dag.Step{}}
	findings := engine.Run(g)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings after disable, got %d", len(findings))
	}
}

func TestEngine_EnableRule(t *testing.T) {
	engine := NewEngine()
	engine.AddRule(&mockRule{
		id:       "TEST001",
		findings: []Finding{{RuleID: "TEST001", Severity: SeverityError, Message: "test"}},
	})
	engine.DisableRule("TEST001")
	engine.EnableRule("TEST001")

	g := &dag.Graph{Nodes: map[string]*dag.Step{}}
	findings := engine.Run(g)
	if len(findings) != 1 {
		t.Errorf("expected 1 finding after re-enable, got %d", len(findings))
	}
}

func TestEngine_SortsBySeverity(t *testing.T) {
	engine := NewEngine()
	engine.AddRule(&mockRule{
		id:       "A_INFO",
		findings: []Finding{{RuleID: "A_INFO", Severity: SeverityInfo, Message: "info"}},
	})
	engine.AddRule(&mockRule{
		id:       "B_ERROR",
		findings: []Finding{{RuleID: "B_ERROR", Severity: SeverityError, Message: "error"}},
	})
	engine.AddRule(&mockRule{
		id:       "C_WARN",
		findings: []Finding{{RuleID: "C_WARN", Severity: SeverityWarning, Message: "warn"}},
	})

	g := &dag.Graph{Nodes: map[string]*dag.Step{}}
	findings := engine.Run(g)
	if len(findings) != 3 {
		t.Fatalf("expected 3 findings, got %d", len(findings))
	}
	if findings[0].Severity != SeverityError {
		t.Errorf("expected first finding to be error, got %v", findings[0].Severity)
	}
	if findings[1].Severity != SeverityWarning {
		t.Errorf("expected second finding to be warning, got %v", findings[1].Severity)
	}
	if findings[2].Severity != SeverityInfo {
		t.Errorf("expected third finding to be info, got %v", findings[2].Severity)
	}
}

func TestEngine_Rules(t *testing.T) {
	engine := NewEngine()
	engine.AddRule(&mockRule{id: "A"})
	engine.AddRule(&mockRule{id: "B"})
	if len(engine.Rules()) != 2 {
		t.Errorf("expected 2 rules, got %d", len(engine.Rules()))
	}
}

func TestHasErrors(t *testing.T) {
	tests := []struct {
		name     string
		findings []Finding
		want     bool
	}{
		{"no findings", nil, false},
		{"only warnings", []Finding{{Severity: SeverityWarning}}, false},
		{"has error", []Finding{{Severity: SeverityError}}, true},
		{"mixed", []Finding{{Severity: SeverityWarning}, {Severity: SeverityError}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasErrors(tt.findings); got != tt.want {
				t.Errorf("HasErrors() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasWarnings(t *testing.T) {
	tests := []struct {
		name     string
		findings []Finding
		want     bool
	}{
		{"no findings", nil, false},
		{"only info", []Finding{{Severity: SeverityInfo}}, false},
		{"has warning", []Finding{{Severity: SeverityWarning}}, true},
		{"has error", []Finding{{Severity: SeverityError}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasWarnings(tt.findings); got != tt.want {
				t.Errorf("HasWarnings() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSummary(t *testing.T) {
	findings := []Finding{
		{Severity: SeverityError},
		{Severity: SeverityError},
		{Severity: SeverityWarning},
		{Severity: SeverityInfo},
	}
	got := Summary(findings)
	want := "2 error(s), 1 warning(s), 1 info(s)"
	if got != want {
		t.Errorf("Summary() = %q, want %q", got, want)
	}
}

func TestSummary_Empty(t *testing.T) {
	got := Summary(nil)
	want := "0 error(s), 0 warning(s), 0 info(s)"
	if got != want {
		t.Errorf("Summary() = %q, want %q", got, want)
	}
}

func TestEngine_MultipleFindings(t *testing.T) {
	engine := NewEngine()
	engine.AddRule(&mockRule{
		id: "MULTI",
		findings: []Finding{
			{RuleID: "MULTI", Severity: SeverityError, Message: "err1"},
			{RuleID: "MULTI", Severity: SeverityWarning, Message: "warn1"},
		},
	})
	g := &dag.Graph{Nodes: map[string]*dag.Step{}}
	findings := engine.Run(g)
	if len(findings) != 2 {
		t.Errorf("expected 2 findings, got %d", len(findings))
	}
}

func TestEngine_NoRules(t *testing.T) {
	engine := NewEngine()
	g := &dag.Graph{Nodes: map[string]*dag.Step{}}
	findings := engine.Run(g)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings, got %d", len(findings))
	}
}

func TestSeverity_MarshalJSON(t *testing.T) {
	tests := []struct {
		s    Severity
		want string
	}{
		{SeverityInfo, `"info"`},
		{SeverityWarning, `"warning"`},
		{SeverityError, `"error"`},
	}
	for _, tt := range tests {
		got, err := tt.s.MarshalJSON()
		if err != nil {
			t.Fatalf("MarshalJSON(%d) error: %v", tt.s, err)
		}
		if string(got) != tt.want {
			t.Errorf("MarshalJSON(%d) = %q, want %q", tt.s, string(got), tt.want)
		}
	}
}

func TestSeverity_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		input string
		want  Severity
	}{
		{`"info"`, SeverityInfo},
		{`"warning"`, SeverityWarning},
		{`"error"`, SeverityError},
		{`0`, SeverityInfo},
		{`1`, SeverityWarning},
		{`2`, SeverityError},
	}
	for _, tt := range tests {
		var s Severity
		err := s.UnmarshalJSON([]byte(tt.input))
		if err != nil {
			t.Fatalf("UnmarshalJSON(%q) error: %v", tt.input, err)
		}
		if s != tt.want {
			t.Errorf("UnmarshalJSON(%q) = %d, want %d", tt.input, s, tt.want)
		}
	}
}

func TestSeverity_UnmarshalJSON_Invalid(t *testing.T) {
	var s Severity
	if err := s.UnmarshalJSON([]byte(`"invalid"`)); err == nil {
		t.Error("expected error for invalid severity string")
	}
	if err := s.UnmarshalJSON([]byte(`99`)); err == nil {
		t.Error("expected error for invalid severity number")
	}
}
