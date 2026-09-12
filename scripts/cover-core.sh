#!/usr/bin/env bash
# cover-core.sh — measure Tier 1 weighted test coverage
# Usage: bash scripts/cover-core.sh [THRESHOLD]  (default threshold: 80)
# Compatible with bash 3.2+ (macOS default)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TIER1_LIST="$SCRIPT_DIR/coverage-tier1.txt"
EXCLUDE_LIST="$SCRIPT_DIR/coverage-exclude.txt"
PROFILE="$REPO_ROOT/coverage-core.out"
FILTERED="$REPO_ROOT/coverage-core-filtered.out"
THRESHOLD="${1:-80}"

# Read Tier 1 packages (strip comments and blank lines) into a single string
PACKAGES=""
while IFS= read -r line; do
    [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]] && continue
    PACKAGES="$PACKAGES $line"
done < "$TIER1_LIST"
PACKAGES="${PACKAGES# }"  # trim leading space

if [[ -z "$PACKAGES" ]]; then
    echo "ERROR: no packages found in $TIER1_LIST" >&2
    exit 1
fi

PKG_COUNT=$(echo "$PACKAGES" | tr ' ' '\n' | wc -l | tr -d ' ')
echo "==> Running Tier 1 coverage ($PKG_COUNT packages)..."
cd "$REPO_ROOT"

# shellcheck disable=SC2086
go test \
    -coverprofile="$PROFILE" \
    -covermode=atomic \
    $PACKAGES 2>&1 | grep -v '^?' | grep -v '^ok' || true

if [[ ! -f "$PROFILE" ]]; then
    echo "ERROR: coverage profile not generated" >&2
    exit 1
fi

# Filter out Tier 3 file-level exclusions
echo "==> Applying file-level exclusions..."
{
    # Always keep the mode header
    head -1 "$PROFILE"
    # Keep lines not matching any exclude pattern
    tail -n +2 "$PROFILE" | while IFS= read -r line; do
        exclude=0
        while IFS= read -r pattern; do
            [[ -z "$pattern" || "$pattern" =~ ^[[:space:]]*# ]] && continue
            if [[ "$line" == *"$pattern"* ]]; then
                exclude=1
                break
            fi
        done < "$EXCLUDE_LIST"
        [[ $exclude -eq 0 ]] && echo "$line"
    done
} > "$FILTERED"

# Per-package summary
echo ""
echo "==> Per-package coverage (Tier 1):"
echo "------------------------------------------------------------"
go tool cover -func="$FILTERED" | grep '^total:' -B9999 | \
    awk '
    /^github\.com/ {
        file = $1
        # Strip function name and line info — extract package
        n = split(file, parts, "/")
        # Remove last segment (file:line)
        pkg = ""
        for (i = 5; i <= n-1; i++) pkg = (pkg == "") ? parts[i] : pkg "/" parts[i]
        pct = $NF
        gsub(/%/, "", pct)
        sum[pkg] += pct
        cnt[pkg]++
    }
    END {
        for (p in sum) printf "  %-55s %6.1f%%\n", p, sum[p]/cnt[p]
    }
    ' | sort
echo "------------------------------------------------------------"

TOTAL=$(go tool cover -func="$FILTERED" | grep '^total:' | awk '{print $NF}' | tr -d '%')
echo ""
printf "  Tier 1 weighted total: %s%%\n" "$TOTAL"

# Integer comparison (strip decimal)
TOTAL_INT="${TOTAL%%.*}"
if [[ "$TOTAL_INT" -ge "$THRESHOLD" ]]; then
    echo "  PASS — meets ${THRESHOLD}% threshold"
    exit 0
else
    echo "  FAIL — below ${THRESHOLD}% threshold (got ${TOTAL}%)"
    exit 1
fi
