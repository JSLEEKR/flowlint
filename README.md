<p align="center">
  <h1 align="center">flowlint</h1>
  <p align="center">Static analyzer and linter for workflow DAG definitions</p>
</p>

<p align="center">
  <a href="https://github.com/JSLEEKR/flowlint/actions"><img src="https://img.shields.io/github/actions/workflow/status/JSLEEKR/flowlint/ci.yml?style=for-the-badge" alt="Build Status"></a>
  <a href="https://github.com/JSLEEKR/flowlint/blob/master/LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue?style=for-the-badge" alt="License"></a>
  <a href="https://goreportcard.com/report/github.com/JSLEEKR/flowlint"><img src="https://img.shields.io/badge/go%20report-A+-brightgreen?style=for-the-badge" alt="Go Report Card"></a>
  <img src="https://img.shields.io/badge/tests-117-success?style=for-the-badge" alt="Tests">
  <img src="https://img.shields.io/badge/go-1.22+-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go Version">
</p>

---

## Why This Exists

Workflow DAGs (in CI/CD pipelines, data pipelines, orchestrators like Airflow/Prefect/Temporal) are defined in YAML/JSON but rarely validated before deployment. Broken workflows — cycles, type mismatches, missing error handlers — only surface at runtime, causing production failures.

**flowlint** catches these issues statically, before your workflow ever runs:

- Detects cycles that would cause infinite loops
- Finds unreachable and orphan steps that waste resources
- Validates type compatibility between connected steps
- Flags missing error handlers and timeout configurations
- Outputs SARIF for CI/CD integration (GitHub Code Scanning, etc.)

## Installation

```bash
go install github.com/JSLEEKR/flowlint/cmd/flowlint@latest
```

Or build from source:

```bash
git clone https://github.com/JSLEEKR/flowlint.git
cd flowlint
go build -o flowlint ./cmd/flowlint/
```

## Quick Start

### Lint a workflow file

```bash
flowlint workflow.yaml
```

### Lint multiple files

```bash
flowlint pipeline.yaml deploy.json staging.yml
```

### JSON output

```bash
flowlint --format json workflow.yaml
```

### SARIF output (for CI/CD)

```bash
flowlint --format sarif workflow.yaml > results.sarif
```

### Strict mode (warnings = errors)

```bash
flowlint --strict workflow.yaml
```

### Disable specific rules

```bash
flowlint --disable ERR001,ERR002 workflow.yaml
```

### List all rules

```bash
flowlint --list-rules
```

## Workflow Format

flowlint accepts YAML or JSON workflow definitions. A workflow consists of named steps with optional dependencies, type contracts, and error handling:

```yaml
name: data-pipeline
version: "1.0"
description: ETL data processing pipeline

steps:
  - id: fetch
    name: Fetch Data
    type: task
    timeout: "60s"
    output:
      name: raw_data
      format: json
    error_handler:
      strategy: retry
      max_retry: 3
      timeout: "10s"

  - id: validate
    name: Validate Schema
    depends_on: [fetch]
    timeout: "30s"
    input:
      name: raw_data
      format: json
    output:
      name: valid_data
      format: object
    error_handler:
      strategy: abort
      timeout: "10s"

  - id: transform
    name: Transform Data
    depends_on: [validate]
    timeout: "120s"
    input:
      name: valid_data
      format: object
    error_handler:
      strategy: fallback
      fallback: notify
      timeout: "30s"

  - id: notify
    name: Send Notification
    depends_on: [validate]
    timeout: "15s"
    error_handler:
      strategy: ignore
      timeout: "5s"
```

Equivalent JSON format:

```json
{
  "name": "data-pipeline",
  "steps": [
    {
      "id": "fetch",
      "timeout": "60s",
      "output": { "name": "raw_data", "format": "json" },
      "error_handler": { "strategy": "retry", "max_retry": 3, "timeout": "10s" }
    },
    {
      "id": "validate",
      "depends_on": ["fetch"],
      "timeout": "30s",
      "input": { "name": "raw_data", "format": "json" },
      "error_handler": { "strategy": "abort", "timeout": "10s" }
    }
  ]
}
```

## Rule Catalog

| Rule ID | Severity | Description |
|---------|----------|-------------|
| `DAG001` | error | **Cycle Detection** -- Detects cycles in workflow DAG using Kahn's algorithm. Cycles cause infinite loops at runtime. |
| `DAG002` | error | **Unreachable Nodes** -- Finds steps not reachable from any root node. These steps will never execute. |
| `DAG003` | warning | **Dead-End Detection** -- Flags leaf steps that produce output but have no downstream consumers. May indicate incomplete wiring. |
| `DAG004` | warning | **Orphan Steps** -- Detects isolated steps with no dependencies and no dependents. Often a copy-paste error. |
| `TYPE001` | error | **Type Compatibility** -- Validates that upstream output formats are compatible with downstream input expectations. Supports json, object, array, string, number, boolean with implicit conversions. |
| `ERR001` | warning | **Error Handler Coverage** -- Flags steps without error handler configuration. All production steps should handle failures. |
| `ERR002` | warning | **Timeout Configuration** -- Flags steps without timeout configuration. Missing timeouts risk resource exhaustion. |
| `ERR003` | warning | **Retry Configuration** -- Validates retry config consistency: retry without handler, retry strategy without max count, fallback strategy without fallback step, references to nonexistent fallback steps (error severity). |

### Type Compatibility Matrix

flowlint validates format compatibility between connected steps. Implicit conversions are allowed for common safe coercions:

| Source Format | Compatible Targets |
|---------------|-------------------|
| `string` | `string` |
| `number` | `number`, `string` |
| `boolean` | `boolean`, `string` |
| `json` | `json`, `object`, `string` |
| `object` | `object`, `json`, `string` |
| `array` | `array`, `json`, `string` |

Unspecified formats (empty string) are treated as universally compatible.

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | No errors (may have warnings/info) |
| `1` | Errors found, or warnings found in `--strict` mode |
| `2` | Usage error (bad flags, no files, invalid format) |

## CI/CD Integration

### GitHub Actions

```yaml
name: Lint Workflows
on: [push, pull_request]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - run: go install github.com/JSLEEKR/flowlint/cmd/flowlint@latest
      - run: flowlint --strict --format sarif workflows/*.yaml > results.sarif
      - uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: results.sarif
```

### Pre-commit Hook

```bash
#!/bin/bash
# .git/hooks/pre-commit
files=$(git diff --cached --name-only --diff-filter=ACM | grep -E '\.(yaml|yml|json)$')
if [ -n "$files" ]; then
  flowlint --strict $files
fi
```

### GitLab CI

```yaml
lint-workflows:
  stage: validate
  image: golang:1.22
  script:
    - go install github.com/JSLEEKR/flowlint/cmd/flowlint@latest
    - flowlint --strict --format json workflows/ > lint-results.json
  artifacts:
    reports:
      codequality: lint-results.json
```

## Output Examples

### Text (default)

```
$ flowlint broken-pipeline.yaml
E error broken-pipeline.yaml: [step: process] type mismatch: step "fetch" output format "json" is incompatible with step "process" input format "number"
E error broken-pipeline.yaml: [step: final] type mismatch: step "process" output format "string" is incompatible with step "final" input format "array"
W warning broken-pipeline.yaml: [step: orphan-step] step "orphan-step" is isolated (no dependencies and no dependents)
W warning broken-pipeline.yaml: [step: fetch] step "fetch" has no error handler configured
W warning broken-pipeline.yaml: [step: fetch] step "fetch" has no timeout configured
```

### JSON

```bash
$ flowlint --format json broken-pipeline.yaml
{
  "file": "broken-pipeline.yaml",
  "findings": [
    {
      "rule_id": "TYPE001",
      "severity": 2,
      "message": "type mismatch: step \"fetch\" output format \"json\" is incompatible with step \"process\" input format \"number\"",
      "step_id": "process"
    }
  ],
  "summary": "2 error(s), 9 warning(s), 0 info(s)"
}
```

### SARIF

```bash
$ flowlint --format sarif broken-pipeline.yaml
{
  "$schema": "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/...",
  "version": "2.1.0",
  "runs": [{
    "tool": { "driver": { "name": "flowlint", "version": "1.0.0" } },
    "results": [...]
  }]
}
```

## Architecture

```
flowlint/
  cmd/flowlint/     CLI entrypoint, flag parsing, exit codes
  pkg/
    parser/          Multi-format parsing (JSON, YAML)
    dag/             DAG data structures, cycle detection (Kahn's), reachability
    lint/            Engine: rule registration, execution, severity sorting
    rules/           Individual rule implementations (modular, pluggable)
    report/          Output formatters (text, JSON, SARIF 2.1.0)
  examples/          Example workflow files
```

### Adding Custom Rules

Implement the `lint.Rule` interface:

```go
type Rule interface {
    ID() string
    Description() string
    DefaultSeverity() Severity
    Check(g *dag.Graph) []Finding
}
```

Register with the engine:

```go
engine := rules.DefaultEngine()
engine.AddRule(&MyCustomRule{})
findings := engine.Run(graph)
```

## Step Schema Reference

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | yes | Unique step identifier |
| `name` | string | no | Human-readable name |
| `type` | string | no | Step type: `task`, `decision`, `parallel`, `wait` (default: `task`) |
| `depends_on` | []string | no | List of upstream step IDs |
| `timeout` | string | no | Step-level timeout (e.g., `"30s"`, `"5m"`) |
| `retry` | int | no | Number of retries |
| `input` | object | no | Input type contract: `{name, format}` |
| `output` | object | no | Output type contract: `{name, format}` |
| `error_handler` | object | no | Error handling config |

### Error Handler Schema

| Field | Type | Description |
|-------|------|-------------|
| `strategy` | string | `retry`, `fallback`, `ignore`, `abort` |
| `max_retry` | int | Maximum retry attempts |
| `timeout` | string | Handler-level timeout |
| `fallback` | string | Fallback step ID (for `fallback` strategy) |

## Dependencies

- `gopkg.in/yaml.v3` -- YAML parsing (only external dependency)
- Go 1.22+ standard library

## License

MIT License - see [LICENSE](LICENSE) for details.
