#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
BENCHMARKS_DIR="$(cd -- "$SCRIPT_DIR/../../.." && pwd)"

cd "$BENCHMARKS_DIR"
SLIM_RUN_BENCHMARK_SUITE=1 go test -count=1 -v ./agntcy-slim/tests --ginkgo.label-filter=benchmark-suite
