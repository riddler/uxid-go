## 1.0.0 / 2026-10-20

The first version of the module: prefixed, K-sortable identifiers a person can
read, copy and route back to their resource, wire-compatible with the Elixir
reference package `uxid`. It is checked against the conformance corpus vendored
from riddler/uxid-ex at the tag `v2.10.0` (`conformance/SOURCE.json` records
the tag, its commit and the corpus hash), and every case in it passes with no
skip list. It builds on Go 1.26 or later, uses the standard library only (the
`go.mod` has no `require`), and keeps no package-level state.

The module path is `github.com/riddler/uxid-go`; install it with
`go get github.com/riddler/uxid-go`. The 2020 module path with a `/v1`
suffix is gone: it was never tagged, and nothing from the 2020 code carries
over.

### Added

- `Generate(opts Options)` builds a time-based id from the prefix, size,
  random width, compact time, case, delimiter and time that `Options` sets.
  The zero `Options` asks for the defaults: no prefix, size xl, standard time,
  lower case, the delimiter `_` and the system clock. The random bytes come
  from `Options.Reader`, an `io.Reader` that defaults to `crypto/rand.Reader`;
  `Options.Rand` fixes them for reproducible test vectors only.
- `Options.From` asks for a deterministic id: the marker `z` (`Z` in upper
  case) and hash characters of a SHA-256 over the prefix and the string, so the
  same inputs always give the same id. It reads neither the byte source nor
  the clock.
- `Size` with `SizeXL` (the zero value), `SizeXS`, `SizeS`, `SizeM` and
  `SizeL`, `Size.String` and `ParseSize`; `Case` with `CaseLower` (the zero
  value) and `CaseUpper`.
- `Decode(id)` and `DecodeWith(id, DecodeOptions)` read an id back to a
  `Decoded` with its `Prefix`, `Size`, `Compact`, `Deterministic` and `Time`.
  `DecodeOptions` carries the `Delimiter` and the decoder's clock `Now`, which
  supplies a compact time's high bits. A string that is not a well-formed id is
  refused with a `*DecodeError` whose `Reason` is `ReasonEmpty`,
  `ReasonInvalidLength` or `ReasonInvalidCharacter`, the corpus names; neither
  function panics on any input.
- `IsValid(id, delimiter, prefix)` and `IsDeterministic(id, delimiter)` answer
  whether a string decodes under the delimiter (and, for `IsValid`, carries
  exactly the prefix, when one is given), and whether it decodes to a
  deterministic body. Neither reads a clock.
- `NewMonotonic(opts Options)` builds a `*Monotonic`, a generator the caller
  holds whose ids increase strictly, also within one millisecond and when the
  clock moves back. `Monotonic.Next` and `Monotonic.NextWith` make the ids;
  a `sync.Mutex` inside the value makes it safe for concurrent use. `Draws`,
  an interface of `Seed` and `Step`, is the injectable source of the random
  values for `NextWith`, for replaying known values only.
- The sentinel errors, matched with `errors.Is`, whose text is the corpus
  name: `ErrCompactTimeOutOfRange`, `ErrRandWidthMismatch`,
  `ErrRandSizeTooLarge`, `ErrFromWithMonotonic`, `ErrRandWithFrom` and
  `ErrRandWithMonotonic`, and this module's own `ErrTimeOutOfRange`,
  `ErrNegativeRandSize`, `ErrUnknownSize`, `ErrUnknownCase` and
  `ErrEmptyRandWithMonotonic`.
- `Version`, the version of the module.
