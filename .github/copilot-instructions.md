# Labtime - Homelab Uptime Monitor

Labtime is a Go-based monitoring system that checks HTTP endpoints,
TLS certificates, and Docker containers, exposing metrics via Prometheus.

## Architecture

The system uses a **generic plugin-based architecture** with three main components:

### 1. Monitor Plugin Pattern

Each monitor type (HTTP, TLS, Docker) implements three interfaces:

- `Target` - configuration data structure
- `MonitorFactory[T, C]` - creates collectors and monitor instances
- `TargetProvider[T]` - extracts targets from YAML config

Example: `internal/monitors/tlsmonitor.go` shows the complete pattern.

### 2. Generic Type System

- `MonitorConfig[T, C]` uses Go generics to eliminate runtime type assertions
- `MonitorFactory[T Target, C prometheus.Collector]` ensures type safety
- All monitors return `Job` interface for unified scheduling

### 3. Dependency Injection for Testing

- Monitor implementations accept function types for external dependencies
- Example: `TLSDialFunc` allows mocking `tls.Dial` in tests
- Use pattern: `DialFunc: func(_, _ string, _ *tls.Config) (*tls.Conn, error)`
  for mocks

## Key Development Patterns

### Monitor Registration

New monitors are registered in `internal/apps/labtime/monitors.go`:

```go
"newtype": &monitorconfig.MonitorConfig[monitors.NewTarget, *prometheus.GaugeVec]{
    Factory:  monitors.NewMonitorFactory{},
    Provider: monitors.NewTargetProvider{},
},
```

### Testing Strategy

- **No real network connections** in unit tests
- Mock external dependencies via function injection
- Example: `TestTLSMonitor_tlsHandshake_DialError` shows mock pattern
- Test files use monitor DTOs instead of inline structs

### DTO Pattern for Configuration

- Each monitor type has its own DTO (Data Transfer Object) in `internal/yamlconfig/`
- DTOs separate configuration structure from business logic
- Pattern: `{MonitorType}MonitorDTO` (e.g., `HTTPMonitorDTO`, `TLSMonitorDTO`)
- Maintains YAML/JSON compatibility while improving code organization
- When adding new monitor types, create corresponding DTO file

### Default Value Management

- **No centralized defaults function** - defaults are applied at the monitor level
- Each monitor's `TargetProvider.GetTargets()` method applies defaults during target
  creation
- **Standard defaults across all monitors:**
  - `Interval`: 60 seconds (when omitted or 0 in YAML)
  - `Name`: Falls back to the primary identifier field when omitted
    - HTTP: Uses `URL` as default name
    - TLS: Uses `Domain` as default name
    - Docker: Uses `ContainerName` as default name
- **Pattern for new monitors:** Check if field is empty/zero value, then assign
  default
- DTOs use `omitempty` tags for optional fields with defaults

### Configuration

- YAML config uses separate DTOs for each monitor type in `internal/yamlconfig/`
- Monitor DTOs: `HTTPMonitorDTO`, `TLSMonitorDTO`, `DockerMonitorDTO`
- No centralized defaults - each monitor's `TargetProvider` applies defaults
  during target creation
- Schema validation via `labtime-configuration-schema.json`
- Example config: `configs/example-config.yaml`
- **Important**: Run `make generate` after modifying `yamlconfig.YamlConfig`
  struct or DTOs to update JSON schema

### Schema Generation

- `cmd/generator/` contains a code generator that reflects the `YamlConfig` struct
- Automatically generates `labtime-configuration-schema.json` for YAML validation
- Uses `github.com/invopop/jsonschema` to create schema from Go struct tags
- Always run `make generate` when adding/modifying config fields

## Development Commands

- `make all` - Full build pipeline (lint, test, generate, build)
- `make generate` - **Required after YamlConfig changes** - Updates JSON schema
- `make test` - Run all tests
- `make lint` - Run golangci-lint
- `go test ./internal/monitors -v -run TestTLS` - Run specific monitor tests

## Project Structure

- `cmd/labtime/` - Main entry point
- `internal/apps/labtime/` - Application setup and HTTP server
- `internal/monitors/` - Monitor implementations (HTTP, TLS, Docker)
- `internal/monitorconfig/` - Generic monitor configuration system
- `internal/scheduler/` - Job scheduling and execution
- `internal/yamlconfig/` - Configuration parsing, defaults, and monitor DTOs
  - `yamlconfig.go` - Main config struct and parsing logic
  - `http_monitor_dto.go` - HTTP monitor configuration DTO
  - `tls_monitor_dto.go` - TLS monitor configuration DTO
  - `docker_monitor_dto.go` - Docker monitor configuration DTO

## Critical Conventions

- All external dependencies must be mockable via function injection
- Prometheus metrics use consistent label patterns: `{type}_monitor_name`, `{type}_{resource}_name`
- Error wrapping: Always use `github.com/pkg/errors.Wrap()` for context
- Logging: Structured with prefixes like `"main:"`, include file/line numbers
