#!/bin/bash

# Performance Benchmark: TUI vs Bubbletea
# Usage: ./run-benchmarks.sh [duration_seconds]

DURATION=${1:-10}

echo "=========================================="
echo "TUI vs Bubbletea Performance Benchmark"
echo "=========================================="
echo "Duration: ${DURATION} seconds per test"
echo ""

# Build both
echo "Building benchmarks..."
go build -o benchmark-tui ./cmd/benchmark-tui/
go build -o benchmark-tea ./cmd/benchmark-tea/
echo ""

# Run TUI benchmark
echo "=========================================="
echo "Running TUI benchmark..."
echo "=========================================="
./benchmark-tui $DURATION

echo ""
sleep 1

# Run Bubbletea benchmark
echo "=========================================="
echo "Running Bubbletea benchmark..."
echo "=========================================="
./benchmark-tea $DURATION

echo ""
echo "=========================================="
echo "Benchmark complete!"
echo "=========================================="

# Cleanup
rm -f benchmark-tui benchmark-tea
