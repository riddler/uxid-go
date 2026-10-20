#!/bin/sh
# Runs scripts/release-guard.sh against fixture values and fails when any
# check passes a value it must refuse or refuses one it must pass. Nothing
# here reaches the network or the repository's own git history: the fixtures
# are files and a throwaway git repository in a temporary directory. The full
# gate runs this (mise.toml, gate:release-guard).
set -eu

here=$(cd "$(dirname "$0")" && pwd)
guard="$here/release-guard.sh"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

failures=0
cases=0

# expect pass|refuse <description> <command...>
expect() {
  want=$1
  what=$2
  shift 2
  cases=$((cases + 1))
  if "$@" >"$tmp/out" 2>"$tmp/err"; then got=pass; else got=refuse; fi
  if [ "$got" != "$want" ]; then
    failures=$((failures + 1))
    echo "FAIL: $what: wanted $want, got $got" >&2
    sed 's/^/  stdout: /' "$tmp/out" >&2
    sed 's/^/  stderr: /' "$tmp/err" >&2
  fi
}

# expect_out <description> <wanted stdout> <command...>
expect_out() {
  what=$1
  want=$2
  shift 2
  cases=$((cases + 1))
  got=$("$@" 2>"$tmp/err") || got="(refused)"
  if [ "$got" != "$want" ]; then
    failures=$((failures + 1))
    echo "FAIL: $what: wanted output '$want', got '$got'" >&2
    sed 's/^/  stderr: /' "$tmp/err" >&2
  fi
}

# --- tag: the tag against `const Version` in a version file.
printf 'package uxid\n\n// Version is the version of this module.\nconst Version = "1.2.3"\n' >"$tmp/version.go"
printf 'package uxid\n' >"$tmp/no-version.go"
printf 'package uxid\n\nconst Version = "1.2.3"\nconst Version = "1.2.4"\n' >"$tmp/two-versions.go"
printf 'package uxid\n\nconst Version = "1.2.3" // trailing\n' >"$tmp/trailing.go"

expect pass "tag matching Version" sh "$guard" tag v1.2.3 "$tmp/version.go"
expect_out "tag prints the version" "1.2.3" sh "$guard" tag v1.2.3 "$tmp/version.go"
expect refuse "tag naming another version" sh "$guard" tag v1.2.4 "$tmp/version.go"
expect refuse "tag without the v" sh "$guard" tag 1.2.3 "$tmp/version.go"
expect refuse "pre-release tag" sh "$guard" tag v1.2.3-rc.1 "$tmp/version.go"
expect refuse "tag with a leading zero" sh "$guard" tag v01.2.3 "$tmp/version.go"
expect refuse "tag with a fourth part" sh "$guard" tag v1.2.3.4 "$tmp/version.go"
expect refuse "version file with no Version" sh "$guard" tag v1.2.3 "$tmp/no-version.go"
expect refuse "version file with two Versions" sh "$guard" tag v1.2.3 "$tmp/two-versions.go"
expect refuse "Version line with a trailing comment" sh "$guard" tag v1.2.3 "$tmp/trailing.go"
expect refuse "missing version file" sh "$guard" tag v1.2.3 "$tmp/absent.go"

# --- module: the path from go.mod.
printf 'module github.com/riddler/uxid-go\n\ngo 1.26\n' >"$tmp/go.mod"
printf 'module github.com/Riddler/uxid-go\n\ngo 1.26\n' >"$tmp/upper.mod"
printf 'go 1.26\n' >"$tmp/nomodule.mod"

expect_out "module path" "github.com/riddler/uxid-go" sh "$guard" module "$tmp/go.mod"
expect refuse "module path needing case encoding" sh "$guard" module "$tmp/upper.mod"
expect refuse "go.mod with no module line" sh "$guard" module "$tmp/nomodule.mod"
expect refuse "missing go.mod" sh "$guard" module "$tmp/absent.mod"

# --- on-branch: ancestry in a throwaway repository.
repo="$tmp/repo"
mkdir "$repo"
g() {
  git -C "$repo" -c user.name=fixture -c user.email=fixture@example.com \
    -c commit.gpgsign=false -c core.hooksPath=/dev/null "$@"
}
g init -q -b main
g commit -q --allow-empty -m first
on_main=$(g rev-parse HEAD)
g checkout -q -b side
g commit -q --allow-empty -m side
off_main=$(g rev-parse HEAD)
g checkout -q main
g commit -q --allow-empty -m second

expect pass "commit on the branch" sh -c "cd '$repo' && sh '$guard' on-branch $on_main main"
expect pass "branch tip itself" sh -c "cd '$repo' && sh '$guard' on-branch main main"
expect refuse "commit off the branch" sh -c "cd '$repo' && sh '$guard' on-branch $off_main main"
expect refuse "unknown branch" sh -c "cd '$repo' && sh '$guard' on-branch $on_main nowhere"
expect refuse "unknown commit" sh -c "cd '$repo' && sh '$guard' on-branch 0123456789abcdef0123456789abcdef01234567 main"

# --- proxy-list: the proxy's answer to @v/list.
: >"$tmp/list-empty"
printf 'v0.9.0\nv1.2.3-rc.1\nv1.2.30\n' >"$tmp/list-other"
printf 'v0.9.0\nv1.2.3\n' >"$tmp/list-with"

expect pass "list unknown to the proxy (404)" sh "$guard" proxy-list 404 v1.2.3 "$tmp/list-empty"
expect pass "list gone from the proxy (410)" sh "$guard" proxy-list 410 v1.2.3 "$tmp/list-empty"
expect pass "empty list" sh "$guard" proxy-list 200 v1.2.3 "$tmp/list-empty"
expect pass "list of other versions only" sh "$guard" proxy-list 200 v1.2.3 "$tmp/list-other"
expect refuse "list already holding the tag" sh "$guard" proxy-list 200 v1.2.3 "$tmp/list-with"
expect refuse "list answered 500" sh "$guard" proxy-list 500 v1.2.3 "$tmp/list-empty"
expect refuse "list request that never answered (000)" sh "$guard" proxy-list 000 v1.2.3 "$tmp/list-empty"
expect refuse "list 200 with no body file" sh "$guard" proxy-list 200 v1.2.3 "$tmp/absent-list"

# --- proxy-info: the proxy's answer to @v/<tag>.info.
printf '{"Version":"v1.2.3","Time":"2026-01-02T03:04:05Z"}' >"$tmp/info-ok"
printf '{\n  "Version": "v1.2.3",\n  "Time": "2026-01-02T03:04:05Z"\n}\n' >"$tmp/info-spaced"
printf '{"Version":"v1.2.30","Time":"2026-01-02T03:04:05Z"}' >"$tmp/info-other"
printf 'not found: unknown revision v1.2.3\n' >"$tmp/info-notfound"

expect pass "info naming the tag" sh "$guard" proxy-info 200 v1.2.3 "$tmp/info-ok"
expect pass "info with whitespace" sh "$guard" proxy-info 200 v1.2.3 "$tmp/info-spaced"
expect refuse "info naming another version" sh "$guard" proxy-info 200 v1.2.3 "$tmp/info-other"
expect refuse "info answered 404" sh "$guard" proxy-info 404 v1.2.3 "$tmp/info-notfound"
expect refuse "info 200 with no body file" sh "$guard" proxy-info 200 v1.2.3 "$tmp/absent-info"

# --- usage.
expect refuse "no subcommand" sh "$guard"
expect refuse "unknown subcommand" sh "$guard" publish

if [ "$failures" -ne 0 ]; then
  echo "release-guard-test: $failures of $cases cases failed" >&2
  exit 1
fi
echo "release-guard-test: $cases cases passed"
