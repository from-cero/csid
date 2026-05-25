#!/usr/bin/env bash
# Benchmark all nodes defined in docker-compose.yml against GET /next-id.
# Runs wrk2 against each node in parallel, then prints a combined summary.
#
# Usage:
#   ./bench.sh [options]
#
# Options (all optional):
#   -d <duration>   test duration, default 30s
#   -R <rate>       target request rate per node, default 5000
#   -t <threads>    threads per node, default 4
#   -c <conns>      connections per node, default 50
#   -s <script>     wrk2 Lua script, default next_id.lua (same dir as this script)
#
# Requirements: wrk2 must be installed and on PATH.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

DURATION="30s"
RATE=5000
THREADS=4
CONNS=50
LUA_SCRIPT="${SCRIPT_DIR}/next_id.lua"

while getopts "d:R:t:c:s:" opt; do
  case $opt in
    d) DURATION="$OPTARG" ;;
    R) RATE="$OPTARG" ;;
    t) THREADS="$OPTARG" ;;
    c) CONNS="$OPTARG" ;;
    s) LUA_SCRIPT="$OPTARG" ;;
    *) echo "Unknown option: $opt" && exit 1 ;;
  esac
done

# Nodes from docker-compose.yml (host port -> container port 8080)
declare -A NODES=(
  [gen-1]="http://localhost:8081"
  [gen-2]="http://localhost:8082"
  [gen-3]="http://localhost:8083"
)

if ! command -v wrk2 &>/dev/null; then
  echo "ERROR: wrk2 not found on PATH. Install from https://github.com/giltene/wrk2"
  exit 1
fi

TMPDIR_RUN="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_RUN"' EXIT

echo "=================================================="
echo " csid benchmark"
echo "  nodes:       ${#NODES[@]}"
echo "  duration:    $DURATION"
echo "  rate/node:   $RATE req/s"
echo "  threads:     $THREADS"
echo "  connections: $CONNS"
echo "  script:      $LUA_SCRIPT"
echo "=================================================="
echo ""

PIDS=()

for node in "${!NODES[@]}"; do
  url="${NODES[$node]}"
  outfile="${TMPDIR_RUN}/${node}.txt"
  echo "Starting wrk2 against $node ($url) ..."
  wrk2 \
    -t "$THREADS" \
    -c "$CONNS" \
    -d "$DURATION" \
    -R "$RATE" \
    --latency \
    -s "$LUA_SCRIPT" \
    "${url}/next-id" \
    > "$outfile" 2>&1 &
  PIDS+=($!)
done

echo ""
echo "All nodes running in parallel, waiting for completion ..."
echo ""

for pid in "${PIDS[@]}"; do
  wait "$pid" || true
done

# -----------------------------------------------------------------------
# Print per-node results and compute combined totals.
# -----------------------------------------------------------------------
total_rps=0
total_reqs=0

echo "=================================================="
echo " Results"
echo "=================================================="
echo ""

for node in "${!NODES[@]}"; do
  outfile="${TMPDIR_RUN}/${node}.txt"
  echo "--- $node (${NODES[$node]}) ---"
  cat "$outfile"

  # Parse "Requests/sec:" from wrk2 summary section (the last occurrence)
  rps_line=$(grep "Requests/sec:" "$outfile" | tail -1 || true)
  if [[ -n "$rps_line" ]]; then
    rps=$(echo "$rps_line" | awk '{print $2}' | tr -d ',')
    total_rps=$(awk "BEGIN { printf \"%.2f\", $total_rps + $rps }")
  fi

  req_line=$(grep "requests in " "$outfile" | tail -1 || true)
  if [[ -n "$req_line" ]]; then
    reqs=$(echo "$req_line" | awk '{print $1}' | tr -d ',')
    total_reqs=$((total_reqs + reqs))
  fi

  echo ""
done

echo "=================================================="
echo " Combined across ${#NODES[@]} nodes"
echo "  Total IDs requested: $total_reqs"
echo "  Combined RPS:        $total_rps req/s"
echo "=================================================="
