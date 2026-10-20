package uxid

import (
	"strings"
	"time"
)

const (
	// compactWindow is the span of a compact time field, 2^40 milliseconds.
	compactWindow = 1 << 40
	// oneDay is how far ahead of the decoder's clock a compact time may be
	// before it is read as one from the window before, in milliseconds.
	oneDay = 86_400_000
)

// Reason is why Decode refused a string: one of the three names of the
// wire format's closed set of decode refusals, which are also the names the
// conformance corpus gives them.
type Reason string

// The three decode refusals, in the order Decode checks them: a body that
// breaks more than one rule is refused for the first.
const (
	// ReasonEmpty refuses an empty body: an empty string, or one that ends
	// in the delimiter.
	ReasonEmpty Reason = "empty"

	// ReasonInvalidLength refuses a body whose length in bytes is not one
	// of the body lengths a producer emits (see DecodeWith).
	ReasonInvalidLength Reason = "invalid_length"

	// ReasonInvalidCharacter refuses a body holding a byte that is not a
	// Crockford Base32 symbol in either case.
	ReasonInvalidCharacter Reason = "invalid_character"
)

// DecodeError is the error Decode and DecodeWith return for a string they
// refuse, always as a *DecodeError: match it with errors.As and read its
// Reason. Its Error text is the reason's name exactly, such as
// "invalid_length".
type DecodeError struct {
	// Reason is the rule the body broke.
	Reason Reason
}

// Error returns the reason's name, the one the conformance corpus gives
// the refusal.
func (e *DecodeError) Error() string {
	return string(e.Reason)
}

// Decoded is what a well-formed id says about itself: its prefix, the size
// and time form its body length gives, whether its body is deterministic,
// and the time it carries. The random field, and a deterministic body's
// hash, are not read back.
type Decoded struct {
	// Prefix is everything before the delimiter the body follows, or the
	// empty string when the id has no delimiter: no prefix. An id that
	// starts with the delimiter, which no producer emits, also has the
	// empty prefix.
	Prefix string

	// Size is the size the body length gives.
	Size Size

	// Compact is whether the body length is a compact-time one (13, 21 or
	// 24). The lengths 10 and 18 are always read as standard time, so a
	// compact xs or compact m id decodes with Compact false.
	Compact bool

	// Deterministic is whether the body starts with the deterministic
	// marker, z or Z. Such a body carries no time.
	Deterministic bool

	// Time is the time the body carries, to the millisecond, in UTC; for a
	// deterministic body it is the zero time.Time. A compact time's high
	// bits come from the decoder's clock (see DecodeWith).
	Time time.Time
}

// DecodeOptions are the inputs of DecodeWith beside the string. The zero
// value asks for the defaults Decode uses: the delimiter "_" and the
// system clock.
type DecodeOptions struct {
	// Delimiter is the string between the prefix and the body. It must be
	// the one the id was generated with, since an id does not record it.
	// The empty string, the zero value, is the delimiter "_".
	Delimiter string

	// Now is the decoder's clock, which supplies a compact time's high
	// bits. The zero time.Time is the current time from the system clock,
	// read only for a compact body. A clock before the Unix epoch is read
	// with the same arithmetic as any other (see DecodeWith).
	Now time.Time
}

// Decode reads an id under the delimiter "_", supplying a compact time's
// high bits from the system clock. It is DecodeWith with the zero
// DecodeOptions; see DecodeWith for the reading and the refusals.
func Decode(id string) (Decoded, error) {
	return DecodeWith(id, DecodeOptions{})
}

// DecodeWith reads an id under opts.Delimiter, supplying a compact time's
// high bits from opts.Now (see DecodeOptions for the defaults).
//
// The body is everything after the last delimiter found reading left to
// right, matches not overlapping, and the prefix everything before it; a
// string with no delimiter is a body with no prefix. That is the
// reference's split: under "--", "usr---01hf7yat00" splits after "usr",
// not at the last "--", and its body starts with "-". The body length in
// bytes gives the size and the time form:
//
//	Body length  Size    Time
//	10           SizeXS  standard
//	13           SizeS   compact
//	14           SizeS   standard
//	18           SizeM   standard
//	21           SizeL   compact
//	22           SizeL   standard
//	24           SizeXL  compact
//	26           SizeXL  standard
//
// The lengths 10 and 18 are also those of a compact xs and a compact m
// body, and they are always read as standard time: such an id decodes to
// its size, Compact false, with a time read from its first 10 characters,
// which is not the time it was generated at.
//
// A body that starts with z or Z is deterministic and carries no time. Its
// size and Compact come from its length like any other body's; a producer
// emits one at a standard length only.
//
// A standard time is the first 10 characters, the first one's low 3 bits
// then nine groups of 5. A compact body carries only the low 40 bits of its
// time in its first 8 characters; the high bits are the clock's, less one
// 2^40 ms window when the result would be more than one day ahead of the
// clock. So the time comes back exact while the id and the clock fall in
// the same 2^40 ms window (about 34.8 years) and the id is no more than one
// day ahead of the clock. With the clock in the first window after the Unix
// epoch, the window before is the one ending at the epoch, so a time read
// from it is before the epoch: Decoded.Time holds such a time. The clock is
// read only for a compact body. A clock before the Unix epoch is in the
// window that ends at the epoch, so under it every compact time is read
// from that window and is before the epoch: the reference's arithmetic,
// whose shift by 40 bits rounds a negative clock down.
//
// A string that is not a well-formed id is refused with a *DecodeError
// whose Reason is the first rule the body breaks, checked in this order:
// ReasonEmpty for an empty body, ReasonInvalidLength for a body whose
// length in bytes is not in the table above, and ReasonInvalidCharacter
// for a body byte that is not a Crockford Base32 symbol in either case.
// Every input gets an answer: DecodeWith never panics.
func DecodeWith(id string, opts DecodeOptions) (Decoded, error) {
	delimiter := opts.Delimiter
	if delimiter == "" {
		delimiter = defaultDelimiter
	}
	prefix, body := splitID(id, delimiter)
	size, compact, reason := form(body)
	if reason != "" {
		return Decoded{}, &DecodeError{Reason: reason}
	}
	decoded := Decoded{Prefix: prefix, Size: size, Compact: compact}
	switch {
	case body[0] == 'z' || body[0] == 'Z':
		decoded.Deterministic = true
	case compact:
		now := opts.Now
		if now.IsZero() {
			now = time.Now()
		}
		decoded.Time = time.UnixMilli(compactTime(readSymbols(body[:8]), now.UnixMilli())).UTC()
	default:
		decoded.Time = time.UnixMilli(int64(readSymbols(body[:1])&7<<45 | readSymbols(body[1:10]))).UTC()
	}
	return decoded, nil
}

// IsValid reports whether id is a well-formed id under delimiter and, when
// prefix is not empty, carries exactly that prefix: whether DecodeWith
// returns no error for it and its Decoded.Prefix is the one asked for. The
// empty prefix accepts any prefix or none; an id with no delimiter has no
// prefix, so it fails any other. The empty delimiter is the delimiter "_",
// as in DecodeOptions. Validity does not depend on the clock, so IsValid
// reads none.
func IsValid(id, delimiter, prefix string) bool {
	decoded, err := DecodeWith(id, DecodeOptions{Delimiter: delimiter, Now: time.UnixMilli(0)})
	return err == nil && (prefix == "" || decoded.Prefix == prefix)
}

// IsDeterministic reports whether id is a well-formed deterministic id
// under delimiter: one DecodeWith returns no error for, whose body starts
// with the marker z or Z. A string that is not well formed is not one,
// whatever its first body character. The empty delimiter is the delimiter
// "_", as in DecodeOptions. It reads no clock.
func IsDeterministic(id, delimiter string) bool {
	decoded, err := DecodeWith(id, DecodeOptions{Delimiter: delimiter, Now: time.UnixMilli(0)})
	return err == nil && decoded.Deterministic
}

// splitID splits id at the last delimiter a left-to-right scan finds, its
// matches not overlapping; with no delimiter the whole id is the body and
// the prefix is empty.
func splitID(id, delimiter string) (prefix, body string) {
	at := -1
	for from := 0; ; {
		i := strings.Index(id[from:], delimiter)
		if i < 0 {
			break
		}
		at = from + i
		from = at + len(delimiter)
	}
	if at < 0 {
		return "", id
	}
	return id[:at], id[at+len(delimiter):]
}

// form is the size and time form of a well-formed body, or the first rule
// it breaks: empty, then its length in bytes, then the alphabet.
func form(body string) (Size, bool, Reason) {
	var size Size
	var compact bool
	switch len(body) {
	case 0:
		return 0, false, ReasonEmpty
	case 10:
		size = SizeXS
	case 13:
		size, compact = SizeS, true
	case 14:
		size = SizeS
	case 18:
		size = SizeM
	case 21:
		size, compact = SizeL, true
	case 22:
		size = SizeL
	case 24:
		size, compact = SizeXL, true
	case 26:
		size = SizeXL
	default:
		return 0, false, ReasonInvalidLength
	}
	for i := 0; i < len(body); i++ {
		if _, ok := symbolValue(body[i]); !ok {
			return 0, false, ReasonInvalidCharacter
		}
	}
	return size, compact, ""
}

// symbolValue is the 5-bit value of a Crockford Base32 symbol in either
// case, and false for any other byte.
func symbolValue(b byte) (uint64, bool) {
	if 'A' <= b && b <= 'Z' {
		b += 'a' - 'A'
	}
	switch {
	case '0' <= b && b <= '9':
		return uint64(b - '0'), true
	case 'a' <= b && b <= 'h':
		return uint64(b-'a') + 10, true
	case b == 'j' || b == 'k':
		return uint64(b-'j') + 18, true
	case b == 'm' || b == 'n':
		return uint64(b-'m') + 20, true
	case 'p' <= b && b <= 't':
		return uint64(b-'p') + 22, true
	case 'v' <= b && b <= 'z':
		return uint64(b-'v') + 27, true
	}
	return 0, false
}

// readSymbols reads symbols as one integer, 5 bits each, most significant
// first. The caller passes symbols only, at most 12 of them.
func readSymbols(symbols string) uint64 {
	var value uint64
	for i := 0; i < len(symbols); i++ {
		v, _ := symbolValue(symbols[i])
		value = value<<5 | v
	}
	return value
}

// compactTime is the time of the compact field low under the clock now, in
// milliseconds: the clock's bits above bit 40 with low, less one window when
// that is more than one day ahead of the clock, which can give a time before
// the epoch. The shift is arithmetic, as the reference's is, so a clock
// before the epoch has the window ending at the epoch as its own.
func compactTime(low uint64, now int64) int64 {
	t := now>>40<<40 | int64(low)
	if t > now && t-now > oneDay {
		t -= compactWindow
	}
	return t
}
