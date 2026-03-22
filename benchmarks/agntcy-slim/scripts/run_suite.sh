#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
BENCHMARKS_DIR="$(cd -- "$SCRIPT_DIR/../.." && pwd)"
GO_TEST_TIMEOUT="${GO_TEST_TIMEOUT:-60m}"

cd "$BENCHMARKS_DIR"
SLIM_RUN_BENCHMARK_SUITE=1 go test -count=1 -timeout "$GO_TEST_TIMEOUT" -v ./agntcy-slim/tests --ginkgo.label-filter=benchmark-suite
