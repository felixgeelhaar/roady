#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT"

COVER_HISTORY=".cover/history.json"
COVER_PROFILE="coverage.out"
mkdir -p $(dirname "$COVER_HISTORY")

if ! command -v coverctl >/dev/null 2>&1; then
  echo "coverctl is required" >&2
  exit 1
fi
if ! command -v relicta >/dev/null 2>&1; then
  echo "relicta is required" >&2
  exit 1
fi

coverctl record --history "$COVER_HISTORY" -p "$COVER_PROFILE"
coverctl check --ratchet --history "$COVER_HISTORY" -p "$COVER_PROFILE"
go test ./...

# Regenerate the latest plan
./cmd/roady plan generate --ai

relicta bump
relicta notes
relicta approve --yes
relicta publish --yes
