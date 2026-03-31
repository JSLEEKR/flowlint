package rules

import "github.com/JSLEEKR/flowlint/pkg/lint"

// AllRules returns all built-in lint rules.
func AllRules() []lint.Rule {
	return []lint.Rule{
		&CycleDetection{},
		&UnreachableNodes{},
		&DeadEndDetection{},
		&OrphanSteps{},
		&TypeCompatibility{},
		&ErrorHandlerCoverage{},
		&TimeoutConfig{},
		&RetryConfig{},
	}
}

// DefaultEngine creates a new lint engine with all built-in rules registered.
func DefaultEngine() *lint.Engine {
	engine := lint.NewEngine()
	for _, r := range AllRules() {
		engine.AddRule(r)
	}
	return engine
}
