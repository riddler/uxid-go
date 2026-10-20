// Package uxid makes prefixed, K-sortable identifiers a person can read, copy
// and route back to their resource, wire-compatible with the Elixir reference
// package uxid.
//
// An id is an optional prefix, a delimiter and a body: a 48-bit millisecond
// time (or a compact 40-bit one) followed by a random field, in Crockford
// Base32, as uxid-ex's wire-format guide lays it out. Generate makes one
// from Options, whose zero value asks for the defaults; its refusals are
// sentinel errors matched with errors.Is.
//
// The package uses the standard library only and takes its random bytes
// from crypto/rand.Reader unless Options names another io.Reader. It
// generates time-based ids; deterministic and monotonic ids, decoding and
// the validity checks are not in it yet.
package uxid
