#!/usr/bin/env bash
# Reviewer-owned isolation: implementation read-only, no external network,
# /tmp backed by owned scratch, copied Go cache, no dependency downloads.
set -u
ROOT=$(pwd)
OWN="$ROOT/.scratch/automation/ci-review"
export PATH="$ROOT/.scratch/automation/venv/bin:$PATH"
export PYTHONDONTWRITEBYTECODE=1
export GOCACHE="$OWN/cache/go-build"
export GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local GOVCS='*:off' GOTELEMETRY=off
export XDG_CACHE_HOME="$OWN/cache" TMPDIR=/tmp/opencode
export OPENAI_API_KEY='' TURSO_AUTH_TOKEN='' TURSO_DATABASE_URL=''
bwrap --ro-bind / / --dev /dev --bind "$OWN" "$OWN" --bind "$OWN/tmp" /tmp \
  --unshare-net --die-with-parent -- "$@"
