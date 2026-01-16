#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT"

COVER_HISTORY=".cover/history.json"
COVER_PROFILE="coverage.out"
EVENTS_FILE=".roady/events.jsonl"
mkdir -p $(dirname "$COVER_HISTORY")

# Helper to log governance events
log_event() {
  local action="$1"
  local actor="${2:-release-script}"
  local metadata="${3:-{}}"
  local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  local id=$(uuidgen 2>/dev/null || cat /proc/sys/kernel/random/uuid 2>/dev/null || echo "evt-$(date +%s)")

  if [ -f "$EVENTS_FILE" ]; then
    echo "{\"id\":\"$id\",\"timestamp\":\"$timestamp\",\"action\":\"$action\",\"actor\":\"$actor\",\"metadata\":$metadata}" >> "$EVENTS_FILE"
  fi
}

echo "=== Roady Release Pipeline ==="
echo ""

# Check dependencies
if ! command -v coverctl >/dev/null 2>&1; then
  echo "coverctl is required" >&2
  exit 1
fi
if ! command -v relicta >/dev/null 2>&1; then
  echo "relicta is required" >&2
  exit 1
fi

# Log release started
log_event "release.started" "release-script" "{\"stage\":\"init\"}"

echo "Step 1: Coverage check..."
coverctl record --history "$COVER_HISTORY" -p "$COVER_PROFILE"
coverctl check --ratchet --history "$COVER_HISTORY" -p "$COVER_PROFILE"
log_event "release.coverage_passed" "coverctl" "{\"profile\":\"$COVER_PROFILE\"}"

echo ""
echo "Step 2: Running tests..."
go test ./...
log_event "release.tests_passed" "go-test" "{}"

echo ""
echo "Step 3: Regenerating plan..."
./roady plan generate --ai 2>/dev/null || ./cmd/roady/main.go plan generate --ai 2>/dev/null || true
log_event "release.plan_generated" "roady" "{}"

echo ""
echo "Step 4: Version bump..."
relicta bump
VERSION=$(relicta status 2>/dev/null | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' | head -1 || echo "unknown")
log_event "release.version_bumped" "relicta" "{\"version\":\"$VERSION\"}"

echo ""
echo "Step 5: Generating release notes..."
relicta notes
log_event "release.notes_generated" "relicta" "{}"

echo ""
echo "Step 6: Approving release..."
relicta approve --yes
log_event "release.approved" "release-script" "{\"version\":\"$VERSION\"}"

echo ""
echo "Step 7: Publishing release..."
relicta publish --yes
log_event "release.published" "relicta" "{\"version\":\"$VERSION\"}"

echo ""
echo "=== Release Complete ==="
log_event "release.completed" "release-script" "{\"version\":\"$VERSION\"}"
