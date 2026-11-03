#!/usr/bin/env bash
# Local gates with existing dependencies only; no installs or module downloads.
set -u
ROOT=$(pwd)
EVIDENCE="$ROOT/.scratch/automation/ci-repair/evidence"
export PATH="$ROOT/.scratch/automation/venv/bin:$PATH"
export PYTHONDONTWRITEBYTECODE=1
export GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local GOVCS='*:off' GOTELEMETRY=off
export OPENAI_API_KEY='' TURSO_AUTH_TOKEN='' TURSO_DATABASE_URL=''
printf '' > "$EVIDENCE/results.txt"
for check in format lint static unit contract fuzz-smoke; do
  timeout 300s make "$check" > "$EVIDENCE/$check.log" 2>&1
  result=$?
  printf '%s exit=%s\n' "$check" "$result" | tee -a "$EVIDENCE/results.txt"
done
