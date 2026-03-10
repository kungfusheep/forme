#!/usr/bin/env bash
set -e

DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$DIR"

tapes=(dashboard fuzzyfinder gtop jumpdashboard todo)

for tape in "${tapes[@]}"; do
  echo "→ recording $tape..."
  vhs "$tape.tape" && echo "  ✓ $tape done" || echo "  ✗ $tape failed"
done

echo ""
echo "output:"
ls -lh *.gif *.mp4 2>/dev/null
