# Implementation Tasks

## Phase 1: Tools Scaffolding & Dockerization (Current)
- [ ] **slim-bench**: Create `Dockerfile` and `go.mod`.
- [ ] **security-probe**: Create `Dockerfile` and `go.mod`.
- [ ] **Build System**: Update `Taskfile.yml` to support building these tool images.
- [ ] **Verification**: Ensure images build successfully.

## Phase 2: Test Runner Enhancements
- [ ] **Network Logic**: Update `testutils/runner` to allow tools to address SLIM nodes (shared network or host networking).
- [ ] **Orchestration**: Add helper functions to start/stop tool containers from within Go tests.

## Phase 3: Performance Benchmarks
- [ ] **Code**: Implement logic in `benchmarks/agntcy-slim/tests/performance_test.go` to invoke `slim-bench`.
- [ ] **Parsing**: Parse `slim-bench` JSON/stdout output into `gmeasure` stats.
- [ ] **Execution**: Run and verify baseline latency numbers.

## Phase 4: Security & Adversarial Tests
- [ ] **Code**: Implement `integrations/agntcy-slim/tests/security_adversarial_test.go` using `security-probe`.
- [ ] **Code**: Implement `integrations/agntcy-slim/tests/adversarial_performance_test.go` (Flood/Slowloris).
- [ ] **Verification**: Ensure tests fail when security controls are disabled (negative testing) and pass when enabled.

## Phase 5: CI Integration
- [ ] **Workflows**: Update GitHub Actions/Taskfiles to run these suites in CI.
