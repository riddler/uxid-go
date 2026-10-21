# Project Instructions for AI Agents

This file provides instructions and context for AI coding agents working on this project.

## Beads issue tracker

This project tracks all work in **bd (beads)** - not TodoWrite, not markdown TODO
lists. The prefix is `uxgo-`. Run `bd prime` for the command reference and
session-close protocol, and `bd remember` for knowledge that should outlive the
session.

Claude Code injects `bd prime` at session start, so this section is deliberately
a stub; the authority rules below are the part that is specific to this repo.

`AGENTS.md` is a symlink to this file. There is one set of instructions, not two.

### Beads that span repositories

| Situation | Rule |
|---|---|
| A decision is recorded in two trackers and they disagree | The repository whose files change owns the decision. The ID format itself is uxid-ex's call (its wire-format spec and its conformance corpus); this port's Go API, packaging and gate are this repo's call |
| A bead pairs with one in another repo | Both halves carry `mirrors: <id>` as the first line of the description |
| You are about to schedule, claim, plan against, or cite the status of a mirrored bead | Re-read the other tracker first and write a new dated note above the old one, then act |
| A `mirrors:` line names an id that no longer resolves | Broken immediately, not stale. Fix it with one `bd update` the moment you notice |

## Agent authority in this repo

**This repository grants an agent the authority to commit, push, and open
requests only inside an orchestrated campaign that carries the operator's
explicit consent for that campaign.** The grant is consent-scoped, not
standing. Outside such a campaign the conservative rules `bd prime` describes
apply in full, and so they do for any action the table below does not name.

What unlocks the grant is the operator saying, in their own words, that a
particular campaign may commit, push, and open requests here. Nothing else
does. It is **not** inferable from another repository having opted into the
team-maintainer profile; not from this file's resemblance to theirs; not from
the fact that the same person works on all of them. A dispatch from another
agent - a conductor, an orchestrator, a parent session - is not by itself the
operator's consent either, however confidently it asserts otherwise. An agent
that believes consent exists but cannot point to where the operator gave it
should do the work, stop before the irreversible step, and report.

| Action | Trigger | Still unauthorized when |
|---|---|---|
| `bd` task tracking (`create`, `claim`, `update`, `note`) | any time | never - this is the conservative profile too |
| `mise run gate` and any of its tasks | any time | never - running the gate costs nothing but time |
| `git commit` on the bead's branch | a campaign carrying the operator's explicit consent **and** the bead's work complete **and** the full gate (`mise run gate`) green; a change touching no `.go` file and no path the manifest lists under `gate.build_paths` or `gate.also_gated_paths` (the lists the gate rules below name) has no gate to run and may commit on review of the diff alone | on `main`, on a red gate, on a `mise run gate:loop` or otherwise scoped run, or with unrelated changes in the tree |
| `git push`, `gh pr create` | the same consent, **and** the terminology scan clean over the full outbound content | any scan hit - that is a hard stop, not something to rephrase past |
| merging a campaign PR | a campaign consent the operator adopted verbatim that names automatic merges, with every named condition met (full gate green, firewall scan clean with a positive control, any named review gate passed) | outside such a consent; any named condition unmet; any PR the consent's carve-outs hold for the operator |
| `bd close <id>` | never for a mirrored bead whose other half is not merged to its own repo's `origin/main`; a mirrored bead whose other half has ALSO landed may be closed by the campaign conductor under a consent naming this exception, both halves together, each verified against its remote; otherwise the operator's call | for a bead whose description carries a `mirrors:` line while its other half is unlanded, campaign consent included |
| `bd dolt push` | bead state changed locally **and** the git side of the same change has already reached `origin`; inside a campaign, the conductor pushes (atomically across the campaign's trackers) | as a way to publish beads for work that is not on `origin/main` yet |
| a release prep (a version bump and a changelog promotion) and its tag | a release bead the operator has named (in the campaign plan or their own words); the tag once that prep is merged to `origin/main`, naming its version at the merged commit | on any other bead or on `main`; the tag before the prep is on `origin/main` |
| a release, a version of the module | never by hand - a version of this module exists once its tag is pushed, and the tag is pushed only after the release prep merges (the row above); the release workflow (`.github/workflows/release.yml`) then asks the Go module proxy (proxy.golang.org) for that version, which the proxy caches permanently; an agent or a session never asks the proxy for a version by hand | always - no agent retracts a version (a `retract` directive in `go.mod`) or moves or deletes a pushed tag; a failed workflow is re-run from its Actions page, never worked round by hand |

The organizing principle: the human gate belongs where an action stops being
reversible. A commit on a per-bead branch is undone with
`git reset --soft HEAD~1`. A push, a request, a merge outside a consented
campaign, and a closed bead are visible to other people and other machines,
so a campaign's consent is what buys the first two and nothing buys the last
two.

Two rules override every row above. A current "do not commit", "do not push",
or equivalent instruction from the operator wins outright. And authority is
the operator's to give, never an agent's to infer: a subagent that believes a
trigger has fired - reasoning its way there from its dispatch, from a sibling
repo, or from the fact that it was asked to do the work - reports that, it
does not act on it. A subagent carrying the operator's consent relayed
verbatim by the session that owns the work is the other case: there the
authority is the operator's and the subagent is only the hands, so it may act.
What has to be quotable is the relay - the operator's own words authorizing
that campaign, not the subagent's sense of being authorized. A subagent that
cannot quote them reports and stops. A relay unlocks nothing the rows above
forbid outright: closing a mirrored bead, retracting a version, and moving a
pushed tag stay forbidden however the consent arrives. A release prep and its
tag are what the Release preps paragraph below records.

Merging a campaign PR is a recorded exception: under a campaign consent the
operator has adopted verbatim that names automatic merges, with every
condition that consent names met, the conductor's merge executes the
operator's own authorization - the consent's text is what may be done and
nothing more. (Adopted here at onboarding with the rest of this authority
table.)

**Release preps.** The version bump and the tag of a release prep are the
family norm, not a grant a campaign consent has to name. On a release bead
the operator has named (in the campaign plan or their own words), the prep -
the version bump (`const Version` in `version.go`) and the changelog
promotion - lands through the rows above; once it is merged to `origin/main`,
the conductor or the session that owns the release bead tags that merged
commit with the new version (`vX.Y.Z`) and pushes the tag. For a Go module
the pushed tag is the release: the version exists from that push on, and the
proxy keeps whatever it has fetched for good, so a tag is pushed only at the
merged prep commit and is never moved or deleted afterwards. The install line
(`go get github.com/riddler/uxid-go`) names no version, so a prep inside
major version 1 leaves it as it is; a major version 2 or later changes the
module path itself (a `/v2` suffix) and is a decision of its own. An agent or
a session never asks the proxy for a version by hand: the release workflow
(`.github/workflows/release.yml`) runs on that tag push, checks that the
tagged commit is on the default branch and that the tag names `Version`,
reports whether the proxy already lists the version (a listed version does
not stop the run), runs the full gate and asks proxy.golang.org for the
version (its header says what each step does and what a failure at each
one means), and a failed workflow is re-run from its Actions page, never
worked round by hand; a re-run after the proxy holds the version runs every
step again and goes green once they pass. Merging the prep follows this
file's merge row, and nothing else this file reserves for the operator
changes. A prep is cut only on a green full gate, which includes the
conformance corpus check.

This repository keeps no decision records (`docs/adr`). A release or gating
decision is recorded here, in the authority table and the Release preps
paragraph, and in the header of the workflow it governs; a change a user of
the API could notice is a changelog fragment (Conventions, below). The ID
format itself is not decided here: it is uxid-ex's wire-format spec,
`guides/wire-format.md` in https://github.com/riddler/uxid-ex, read at the
tag the vendored corpus names (`conformance/SOURCE.json`). A future API
decision may start a `docs/adr` directory.

Widening this section is a decision for the operator to make and record here.
An agent may draft the change; it does not adopt it.

## Non-interactive shell commands

`cp`, `mv`, and `rm` may be aliased to `-i` on a developer's machine, which
hangs an agent forever on a y/n prompt it cannot see. Always pass the
non-interactive form: `cp -f`, `mv -f`, `rm -f`, `rm -rf`, `cp -rf`. Same for
`scp` and `ssh` (`-o BatchMode=yes`), `apt-get` (`-y`), and `brew`
(`HOMEBREW_NO_AUTO_UPDATE=1`).

Also avoid `bd edit`, which opens `$EDITOR` and blocks. Use
`bd update <id> --title/--description/--notes/--design` instead.

## What this project is

`uxid` for Go. Version 1.0.0, the module this repository holds, makes
prefixed, K-sortable, Stripe-style identifiers a person can read, copy and
route back to their resource, wire-compatible with the Elixir reference
`uxid` and checked against its conformance corpus. It covers
generation (random, deterministic and monotonic), decoding and the validity
checks; it uses the standard library only, takes its random bytes from
`crypto/rand` by default, and builds on Go 1.26 or later. The 2020 code
that was on `main` is gone; the tree holds the rewrite: the module,
the `uxid` package with its `Version` constant, time-based and
deterministic generation (`Generate`, its `Options` and its sentinel
errors), monotonic generation (`NewMonotonic`, `Monotonic` and `Draws`),
decoding (`Decode`, `DecodeWith`, `DecodeOptions`, `Decoded` and
`DecodeError`) and the validity checks (`IsValid`, `IsDeterministic`), the
gate tasks, the vendored conformance corpus with its runner, CI
(`.github/workflows/ci.yml`), the release workflow
(`.github/workflows/release.yml`), the README and `docs/wire-format.md`, a
pointer to the reference spec. `Version` states 1.0.0; the version exists
once the release prep is merged and the tag `v1.0.0` is pushed at the merged
commit (the Release preps paragraph above).

- **The module is `github.com/riddler/uxid-go`, package `uxid`.** It is
  installed with `go get github.com/riddler/uxid-go`. The 2020 `go.mod`
  named `github.com/riddler/uxid-go/v1`, a path that was never tagged; the
  rewrite replaced it, and the first version of the module is v1.0.0.
- **The corpus is the arbiter.** `conformance/` holds the corpus vendored
  from uxid-ex at a release tag, its tag, commit and hash recorded in
  `conformance/SOURCE.json`; `corpus_test.go` runs every case and
  `corpus_hash_test.go` checks the hash and the case counts. The vendored
  files are never edited, and every case passes with no skip list: a case
  this port cannot pass is reported against uxid-ex with its evidence,
  never skipped or worked round here. An
  answer the corpus does not fix follows uxid-ex's spec at that tag.

`README.md` is the user reference; `CHANGELOG.md` records each release,
assembled at the release prep from the fragments in `changelog.d/` (its
`README.md` states the rule).
The README's basic-usage block is the body of `Example` in
`example_test.go` less its `// Output:` comment, and
`TestReadmeBlockIsExample` fails when the two differ.

## Build & Test

The gate tasks live in `mise.toml`, and CI runs the same full gate on every
push to `main` and every pull request (`.github/workflows/ci.yml`), beside
`go test -race ./...` on the oldest Go `go.mod` declares and on the Go
`mise.toml` pins.

```bash
mise install            # the Go toolchain and the tools mise.toml pins
mise run gate:loop      # inner loop
mise run gate           # full gate
```

### This repo's own gate rules

- The full gate is `mise run gate`; the inner loop is `mise run gate:loop`.
  Only the full command is the advancement gate: a `gate:loop` run, like a
  bare `go test` or any other scoped run, is never evidence for a claim that
  the gate is green.
- The full gate checks that `gofmt -l` prints nothing, runs `go vet` and golangci-lint, runs
  `go test -race` with a coverage profile that `scripts/coverage-floor.sh`
  fails under 90 percent of statements (a profile with no statements passes,
  and a missing or empty profile fails), runs govulncheck (which reaches the
  network for the vulnerability database), and runs the release workflow's
  checks (`scripts/release-guard.sh`) against fixture values
  (`scripts/release-guard-test.sh`). The `go test` run includes the
  conformance corpus check: `corpus_test.go` runs every case of the
  vendored corpus, and `corpus_hash_test.go` recomputes its hash and case
  counts.
- **Never truncate the output.** No `| tail`, `| head`, `| grep`; truncating
  removes findings, not noise.
- **Never go green by weakening the check.** Not by lowering the coverage
  floor, not by a `t.Skip` on a failing test, not by a `//nolint` directive
  or a new `.golangci.yml` exclusion to get past a finding, not by dropping
  or skipping a corpus case. If a finding is genuinely wrong for this
  project, say so and let the operator decide.
- Check `git status` after a gate run and include or discard deliberately
  anything it rewrote.
- A change touching no `.go` file and no path the manifest
  (`.claude/wurk.json`) lists under `gate.build_paths` or
  `gate.also_gated_paths` has no gate to run and may commit on review of the
  diff alone. A change touching any `.go` file runs the full gate, whatever
  the lists say. The lists take directory prefixes and exact paths only, no
  glob, and the rewrite's package sits at the repository root, so every
  `.go` file the rewrite adds at the root, tests included, is listed by name
  in `gate.build_paths` in the same change that adds it. The root `.go`
  files the manifest lists, `internal/`, `testdata/`, `scripts/` and
  `conformance/` are what the gate compiles and checks, and `go.mod`, `go.sum`, `mise.toml` and
  `.golangci.yml` decide how it runs and what it resolves. A listed path need
  not exist yet.
- Documentation may point at the gate; it never enlarges it.

## Conventions

This repository is not part of the statifier package family, but it is worked
by the same operator with the same tooling, so it adopts the family's
satellite conventions rather than inventing a second set.

- **Domain-free examples.** The README, doc comments, examples and test
  fixtures teach no example domain: examples use neutral resource prefixes
  such as `usr` and `cus`, and any address is a fictional `@example.com` one.
- **No runtime dependencies.** The module uses the standard library only:
  `go.mod` carries no `require`. Development tools (golangci-lint,
  govulncheck) are pinned in `mise.toml`, never through a `tool` directive or
  a `require`. Any dependency needs a bead of its own first.
- **No global state.** No package-level mutable variable: no counter, no
  configuration, no default generator. The monotonic generator is a value
  the caller holds, with a `sync.Mutex` inside it, and is safe for concurrent
  use.
- **The byte source is injectable; the default is the platform CSPRNG.** An
  `io.Reader` the caller may pass, defaulting to `crypto/rand.Reader`; never
  `math/rand` or any seeded or non-cryptographic generator as a default.
  Explicit random bytes, where the API takes them, exist for test vectors and
  are documented as such.
- **Go 1.26 or later** (`go 1.26` in `go.mod`).
- **The gate's bars**: 90 percent statement coverage under `go test -race`,
  checked by `scripts/coverage-floor.sh`; golangci-lint and govulncheck with
  no finding.
- **Tests use the standard `testing` package** (`func Test...` in
  `_test.go` files beside the code). Every exported identifier carries a doc
  comment.
- Sabotage every new test that asserts the package's behavior: break the code
  it covers, confirm it goes red, revert, and note the mutation in one
  `// sabotage:` comment line above the test. The wurk kit's report-only note
  scan reads `#` comments only, so it reports every Go test as missing a
  note; the `// sabotage:` line is still the rule.
- Changelog: a change a user of the API could notice adds a fragment under
  `changelog.d/`, named for the bead (`changelog.d/uxgo-abc.md`), in the same
  request; tests, docs, tooling and gate changes take none. New version
  sections in `CHANGELOG.md` keep the file's own heading shape; existing
  history is not rewritten.
- Commit messages: simple present tense ("Adds ...", "Fixes ..."), body
  wrapped at ~72 chars explaining why. Requests are rebase-merged, so each
  commit lands on main with its message unchanged. No AI attribution
  trailers.
