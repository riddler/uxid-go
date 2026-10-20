#!/bin/sh
# Fails when the statement coverage in a Go coverage profile is under the
# floor (90 percent), or when the profile is missing, empty or unreadable.
# Usage: scripts/coverage-floor.sh [profile]   (default: coverage.out)
#
# A package with no statements has nothing to cover: the profile then holds
# its mode line only, and that is reported as such and passes. Once the
# package holds a statement the floor applies in full.
set -eu

floor=90
profile="${1:-coverage.out}"

if [ ! -s "$profile" ]; then
  echo "coverage-floor: profile '$profile' is missing or empty" >&2
  exit 1
fi

case "$(head -n 1 "$profile")" in
  mode:*) ;;
  *)
    echo "coverage-floor: '$profile' is not a Go coverage profile" >&2
    exit 1
    ;;
esac

# Each profile line after the mode line: file:range numStmts count.
result=$(awk 'NR > 1 { total += $2; if ($3 > 0) covered += $2 }
  END { printf "%d %d\n", total + 0, covered + 0 }' "$profile")
total=${result% *}
covered=${result#* }

if [ "$total" -eq 0 ]; then
  echo "coverage-floor: no statements in the profile, nothing to cover"
  exit 0
fi

pct=$(awk -v c="$covered" -v t="$total" 'BEGIN { printf "%.1f", c * 100 / t }')
if awk -v c="$covered" -v t="$total" -v f="$floor" 'BEGIN { exit !(c * 100 < f * t) }'; then
  echo "coverage-floor: ${pct}% of statements is under the ${floor}% floor" >&2
  exit 1
fi
echo "coverage-floor: ${pct}% of statements (floor ${floor}%)"
