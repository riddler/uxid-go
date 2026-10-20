#!/bin/sh
# The release workflow's checks (.github/workflows/release.yml), one
# subcommand each. Every subcommand decides from the values it is given and
# reaches no network: the workflow fetches, this script judges, so each
# check can be run against fixture values (scripts/release-guard-test.sh
# does, inside the full gate). Exit 0 means the check passed; any other exit
# means it refused, with the reason on stderr.
#
#   release-guard.sh tag <tag> [version-file]
#       The tag is vMAJOR.MINOR.PATCH and names the version that
#       `const Version` states in the version file (default: version.go).
#       The file is read as text, never compiled. Prints the version.
#   release-guard.sh module [go-mod]
#       Prints the module path from the go.mod file (default: go.mod), and
#       refuses one the proxy would need case-encoded or escaped.
#   release-guard.sh on-branch <commit> <branch-ref>
#       The commit is an ancestor of (or is) the branch ref, in the git
#       repository of the current directory.
#   release-guard.sh proxy-list <http-status> <tag> <body-file>
#       Judges the proxy's answer to `@v/list`: 404 or 410 (the proxy knows
#       no version of the module) and a 200 list without the tag pass; a 200
#       list holding the tag refuses (the version is already there); any
#       other status refuses.
#   release-guard.sh proxy-info <http-status> <tag> <body-file>
#       Judges the proxy's answer to `@v/<tag>.info`: passes only on a 200
#       whose JSON names the tag as its Version.
set -eu

fail() {
  if [ "${GITHUB_ACTIONS:-}" = "true" ]; then
    echo "::error::$*" >&2
  else
    echo "release-guard: $*" >&2
  fi
  exit 1
}

usage() {
  fail "usage: release-guard.sh tag|module|on-branch|proxy-list|proxy-info ..."
}

semver_tag() {
  printf '%s\n' "$1" | grep -Eq '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$'
}

cmd_tag() {
  [ "$#" -ge 1 ] || usage
  tag=$1
  file=${2:-version.go}
  semver_tag "$tag" || fail "tag '$tag' is not vMAJOR.MINOR.PATCH"
  [ -f "$file" ] || fail "no $file to read Version from"
  count=$(grep -Ec '^const Version = "[^"]*"$' "$file" || true)
  [ "$count" -eq 1 ] || fail "$file holds $count lines of the form const Version = \"...\", not exactly one"
  version=$(sed -n 's/^const Version = "\([^"]*\)"$/\1/p' "$file")
  if [ "${tag#v}" != "$version" ]; then
    fail "tag $tag names ${tag#v} but $file says Version = \"$version\""
  fi
  echo "$version"
}

cmd_module() {
  file=${1:-go.mod}
  [ -f "$file" ] || fail "no $file to read the module path from"
  module=$(awk '$1 == "module" { print $2; exit }' "$file")
  [ -n "$module" ] || fail "$file names no module"
  # The proxy protocol case-encodes upper-case letters; a path of lower-case
  # letters, digits and . _ - / needs no encoding and goes into a URL as is.
  printf '%s\n' "$module" | grep -Eq '^[a-z0-9._/-]+$' ||
    fail "module path '$module' needs encoding for the proxy; this script does not encode"
  echo "$module"
}

cmd_on_branch() {
  [ "$#" -eq 2 ] || usage
  commit=$1
  ref=$2
  git rev-parse -q --verify "$commit^{commit}" >/dev/null || fail "$commit is not a commit here"
  git rev-parse -q --verify "$ref^{commit}" >/dev/null || fail "$ref is not a branch here"
  git merge-base --is-ancestor "$commit" "$ref" || fail "$commit is not on $ref"
}

cmd_proxy_list() {
  [ "$#" -eq 3 ] || usage
  status=$1
  tag=$2
  body=$3
  case "$status" in
    404 | 410) ;;
    200)
      [ -f "$body" ] || fail "no body file $body for the proxy's 200 answer"
      if grep -Fxq -- "$tag" "$body"; then
        fail "the proxy already lists $tag; reported, and not requested again"
      fi
      ;;
    *) fail "the proxy answered $status to the version list" ;;
  esac
}

cmd_proxy_info() {
  [ "$#" -eq 3 ] || usage
  status=$1
  tag=$2
  body=$3
  [ "$status" = "200" ] || fail "the proxy answered $status for $tag"
  [ -f "$body" ] || fail "no body file $body for the proxy's 200 answer"
  tr -d ' \t\r\n' <"$body" | grep -Fq "\"Version\":\"$tag\"" ||
    fail "the proxy's 200 answer does not name $tag as its Version"
}

[ "$#" -ge 1 ] || usage
sub=$1
shift
case "$sub" in
  tag) cmd_tag "$@" ;;
  module) cmd_module "$@" ;;
  on-branch) cmd_on_branch "$@" ;;
  proxy-list) cmd_proxy_list "$@" ;;
  proxy-info) cmd_proxy_info "$@" ;;
  *) usage ;;
esac
