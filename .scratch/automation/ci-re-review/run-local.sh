#!/usr/bin/env bash
set -u
ROOT=$(pwd)
OWN="$ROOT/.scratch/automation/ci-re-review"
export PATH="$ROOT/.scratch/automation/venv/bin:$PATH"
export PYTHONDONTWRITEBYTECODE=1 GOCACHE="$OWN/cache/go-build"
export GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local GOVCS='*:off' GOTELEMETRY=off
export XDG_CACHE_HOME="$OWN/cache" TMPDIR=/tmp/opencode
export OPENAI_API_KEY='' TURSO_AUTH_TOKEN='' TURSO_DATABASE_URL=''
exec bwrap --ro-bind / / --dev /dev --bind "$OWN" "$OWN" --bind "$OWN/tmp" /tmp --unshare-net --die-with-parent -- "$@"
