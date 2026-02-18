#!/usr/bin/env bash
# Point git to the project's .githooks directory.
# Run once after cloning: ./scripts/setup-hooks.sh
set -euo pipefail
git config core.hooksPath .githooks
echo "Git hooks activated (.githooks/)"
