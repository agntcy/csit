#!/bin/bash
set -e

# Configuration
SIZES=(16 128 1024 10240)
CLIENTS=(1 10 50)
MODES=("request" "pub")
DURATION="5s"
OUTPUT_DIR="reports"

# Setup
mkdir -p "$OUTPUT_DIR"
SUMMARY_FILE="$OUTPUT_DIR/suite_summary.md"

echo "# SLIM Benchmark Suite Report" > "$SUMMARY_FILE"
echo "**Date:** $(date)" >> "$SUMMARY_FILE"
echo "" >> "$SUMMARY_FILE"

# Build the tool first for consistency
echo "Building slim-bench..."
go build -o slim-bench main.go

# Execution Loop
for mode in "${MODES[@]}"; do
  for clients in "${CLIENTS[@]}"; do
    for size in "${SIZES[@]}"; do
      # For pub mode, we want to test throughout, so let's uncap the rate or set it very high
      RATE_ARG=""
      if [ "$mode" == "pub" ]; then
          RATE_ARG="--rate=0" # Max speed
      else
          RATE_ARG="--rate=1000" # Paced for latency stats
      fi

      echo "----------------------------------------------------------------"
      echo "Running benchmark: Mode=${mode}, Clients=${clients}, Size=${size}B"

      REPORT_FILE="$OUTPUT_DIR/report_${mode}_c${clients}_s${size}.md"

      # Run the benchmark
      ./slim-bench \
          --mode=$mode \
          --clients=$clients \
          --size=$size \
          $RATE_ARG \
          --duration=$DURATION \
          --output="$REPORT_FILE"

      # Append to summary
      echo "## Benchmark: Mode=${mode}, Clients=${clients}, Size=${size}B" >> "$SUMMARY_FILE"

      if [ -f "$REPORT_FILE" ]; then
          # Skip header lines
          tail -n +5 "$REPORT_FILE" >> "$SUMMARY_FILE"
      else
          echo "Error: Report file not generated." >> "$SUMMARY_FILE"
      fi

      echo "" >> "$SUMMARY_FILE"
      echo "---" >> "$SUMMARY_FILE"
    done
  done
done

echo "----------------------------------------------------------------"
echo "Benchmark suite completed."
echo "Summary report generated at: $SUMMARY_FILE"
