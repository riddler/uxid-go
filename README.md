# UXID

[![MIT License][badge_license_url]](LICENSE)
[![CI][badge_ci_url]](https://github.com/riddler/uxid-go/actions/workflows/ci.yml)

User eXperience focused IDentifiers for Go: prefixed, K-sortable,
Stripe-style ids a person can read, copy and route back to their resource. An
id looks like `usr_01hf7yat0004h1`: the prefix names the resource, and the
body carries a time and randomness in lower-case Crockford Base32. The ids are
wire-compatible with the Elixir reference implementation,
[uxid-ex](https://github.com/riddler/uxid-ex), and the module is checked
against that implementation's conformance corpus.

## Why UXID

An auto-increment key leaks how many rows a table holds and invites
enumeration; a random UUID fixes both but is long, unordered and anonymous, so
in a log line, a URL or a support thread nobody can tell what it points at, and
a double-click selects only part of it. A UXID names its resource in its
prefix, selects whole on a double-click, reads aloud without ambiguity, sorts
by creation time so it indexes well, and is generated in the application with
no coordination between nodes. Its size is tunable for low-cardinality
resources, a monotonic generator keeps a burst within one millisecond unique
and in order, and a deterministic mode maps the same input to the same id.

Because the format is shared, an id this module makes is read by the other
implementations of the format, and the other way round.

## Install

```bash
go get github.com/riddler/uxid-go
```

The import path is `github.com/riddler/uxid-go` and the package is named
`uxid`.

## Basic usage

```go
// A prefix names the resource; the body is a time and randomness.
id, err := uxid.Generate(uxid.Options{Prefix: "usr"})
if err != nil {
	panic(err)
}
fmt.Println(strings.HasPrefix(id, "usr_"), len(id)) // true 30

// A size shrinks the random part for low-cardinality resources, and a fixed
// time and fixed random bytes reproduce an id exactly. Fixed bytes are for
// test vectors only, never a source of randomness.
short, err := uxid.Generate(uxid.Options{
	Prefix: "cus",
	Size:   uxid.SizeS,
	Time:   time.UnixMilli(1_700_000_000_000),
	Rand:   []byte{0x01, 0x23},
})
if err != nil {
	panic(err)
}
fmt.Println(short) // cus_01hf7yat0004h1

// Deterministic: the same prefix and input give the same id, every time.
named, err := uxid.Generate(uxid.Options{Prefix: "usr", From: new("alice@example.com")})
if err != nil {
	panic(err)
}
fmt.Println(named, uxid.IsDeterministic(named, "_")) // usr_zcvt7epac0t1ebcsjfyf7cwz25 true

// Read an id back. A malformed id is a *DecodeError, matched with errors.As.
decoded, err := uxid.Decode(short)
if err != nil {
	panic(err)
}
fmt.Println(decoded.Prefix, decoded.Size, decoded.Time.UnixMilli())           // cus s 1700000000000
fmt.Println(uxid.IsValid(short, "_", "cus"), uxid.IsValid(short, "_", "usr")) // true false
_, err = uxid.Decode("cus_")
fmt.Println(err) // empty

// Monotonic: ids from one generator increase strictly, also within one
// millisecond. You hold the generator; the module keeps no global state.
ids, err := uxid.NewMonotonic(uxid.Options{Prefix: "usr", Size: uxid.SizeM})
if err != nil {
	panic(err)
}
first, err := ids.Next()
if err != nil {
	panic(err)
}
second, err := ids.Next()
if err != nil {
	panic(err)
}
fmt.Println(first < second) // true
```

The block above needs `fmt`, `strings` and `time` beside the module's import.
It also runs as a testable example (`Example` in `example_test.go`), whose
output `go test` checks, and a test keeps the two identical.

A request the wire format refuses, such as a random width above 10 bytes,
returns a sentinel error such as `uxid.ErrRandSizeTooLarge`: match it with
`errors.Is`, or read its text, a stable `snake_case` name, rather than
matching on anything else. A string that is not an id is a `*uxid.DecodeError`
whose `Reason` names the rule it broke.

`Generate`, `Decode` and a monotonic generator read the system clock when they
need a time, and `Generate` and the generator read random bytes from
`crypto/rand.Reader` when they need them. `Options.Time` and
`DecodeOptions.Now` take the time instead, `Options.Reader` takes the byte
source, and `Monotonic.NextWith` takes the time and the draws of one call.
Fixed random bytes (`Options.Rand`) exist for test vectors: an id built from
them is exactly as guessable as those bytes.

## Compatibility

- **Go 1.26 or later** (the `go` line in `go.mod`). CI runs the tests on that
  Go and on the Go `mise.toml` pins, and the full quality gate on the
  pinned one.
- **Standard library only.** `go.mod` has no `require`. Random bytes come from
  `crypto/rand.Reader` unless you give `Options.Reader`, and the module never
  uses `math/rand`.
- **Concurrency.** One `*uxid.Monotonic` may be shared across goroutines: a
  `sync.Mutex` inside it orders the calls, and ids from it increase strictly in
  the order the calls take the lock. Nothing is kept at package level. Ids from
  two generators carry no order between them within a millisecond. `Generate`,
  `Decode` and the validity checks hold no state and are safe from any
  goroutine, provided an `io.Reader` you pass as `Options.Reader` is.

## Documentation

- Look up
  - [The API reference on pkg.go.dev](https://pkg.go.dev/github.com/riddler/uxid-go):
    every exported identifier, each with a doc comment, and the runnable
    examples.
  - [The wire format](https://github.com/riddler/uxid-go/blob/main/docs/wire-format.md):
    where the normative layout of an id lives, and which version of it this
    module follows.
  - [The changelog](https://github.com/riddler/uxid-go/blob/main/CHANGELOG.md):
    what changed in each version.
  - [The conformance corpus](https://github.com/riddler/uxid-go/tree/main/conformance):
    the cases every implementation of the wire format passes, vendored here
    from uxid-ex.
- Understand
  - [The reference spec, at v2.10.0][guide_wire_format_url]: the normative
    layout of an id - alphabet, bit packing, sizes, prefix grammar,
    deterministic and monotonic bodies, and decoding.
  - [Why a UXID has a prefix, a time and randomness][guide_why_url]: what each
    part of an id is for, and what you trade when you tune it.
  - [Why monotonic mode exists and what it costs][guide_monotonic_url]:
    same-millisecond uniqueness and ordering, and the security tradeoff.
  - [_Designing APIs for humans: object IDs_][stripe_ids_url]: the Stripe id
    design many of UXID's choices follow.

## Development

After checking out the repository, run `mise install` for the Go toolchain and
the tools `mise.toml` pins. Then `mise run gate:loop` runs the inner loop, and
`mise run gate` runs the full quality gate: formatting, `go vet`,
golangci-lint, `go test -race` with a coverage floor of 90 percent,
govulncheck, and the release workflow's checks run against fixture values.
The tests include the conformance corpus, every case with no skip list.

A release is a tag pushed at the merged release-prep commit, never made from a
working checkout. The tag push runs `.github/workflows/release.yml`, which
checks that the commit is on the default branch and that the tag names the
version in `version.go`, runs the full gate and asks the Go module proxy for
the version.

## Contributing

Bug reports and pull requests are welcome on GitHub at
https://github.com/riddler/uxid-go.

## License

The module is available as open source under the terms of the
[MIT License](https://opensource.org/licenses/MIT).

<!-- LINKS -->
[guide_wire_format_url]: https://github.com/riddler/uxid-ex/blob/v2.10.0/guides/wire-format.md
[guide_why_url]: https://github.com/riddler/uxid-ex/blob/v2.10.0/guides/why-prefix-time-randomness.md
[guide_monotonic_url]: https://github.com/riddler/uxid-ex/blob/v2.10.0/guides/monotonic.md
[stripe_ids_url]: https://dev.to/stripe/designing-apis-for-humans-object-ids-3o5a

<!-- BADGES -->
[badge_license_url]: https://img.shields.io/badge/license-MIT-brightgreen.svg
[badge_ci_url]: https://github.com/riddler/uxid-go/actions/workflows/ci.yml/badge.svg
