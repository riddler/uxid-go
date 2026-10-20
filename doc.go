// Package uxid makes prefixed, K-sortable identifiers a person can read, copy
// and route back to their resource, wire-compatible with the Elixir reference
// package uxid.
//
// An id is an optional prefix, a delimiter and a body: a 48-bit millisecond
// time (or a compact 40-bit one) followed by a random field, in Crockford
// Base32, as uxid-ex's wire-format guide lays it out. Given Options.From,
// the body is deterministic instead: the marker z (Z in upper case) and
// characters of a SHA-256 hash of the prefix and that string, so the same
// inputs always give the same id. Generate makes an id from Options, whose
// zero value asks for the defaults; its refusals are sentinel errors
// matched with errors.Is.
//
// Decode and DecodeWith read an id back to its prefix, size, time form and
// time, a compact time's high bits coming from a clock DecodeOptions may
// fix; a string that is not a well-formed id is refused with a
// *DecodeError naming the reason, matched with errors.As. IsValid and
// IsDeterministic answer whether a string decodes, and whether it decodes
// to a deterministic id, under a given delimiter.
//
// The package uses the standard library only and takes its random bytes
// from crypto/rand.Reader unless Options names another io.Reader. It
// generates time-based and deterministic ids and decodes them; monotonic
// ids are not in it yet.
package uxid
