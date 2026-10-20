package uxid

import (
	"io"
	"strconv"
	"time"
)

// Size is an id's size, which fixes the random field's width, and so the
// body length, for standard and compact time.
//
//	Size    Random bytes        Body length
//	        standard  compact   standard  compact
//	SizeXS  0         1         10        10
//	SizeS   2         3         14        13
//	SizeM   5         6         18        18
//	SizeL   7         8         22        21
//	SizeXL  10        10        26        24
//
// The zero value is SizeXL, the default size. A Size outside the five
// constants is refused with ErrUnknownSize.
type Size int

// The five sizes. SizeXL is listed first so that it is the zero value.
const (
	// SizeXL is extra large, the default: 10 random bytes.
	SizeXL Size = iota
	// SizeXS is extra small: no random field in standard time.
	SizeXS
	// SizeS is small.
	SizeS
	// SizeM is medium.
	SizeM
	// SizeL is large.
	SizeL
)

// ParseSize reads a size from its short spelling (xs, s, m, l, xl) or its
// long one (xsmall, small, medium, large, xlarge), in lower case exactly.
// Any other string is ErrUnknownSize.
func ParseSize(text string) (Size, error) {
	switch text {
	case "xs", "xsmall":
		return SizeXS, nil
	case "s", "small":
		return SizeS, nil
	case "m", "medium":
		return SizeM, nil
	case "l", "large":
		return SizeL, nil
	case "xl", "xlarge":
		return SizeXL, nil
	}
	return 0, ErrUnknownSize
}

// String returns the size's short spelling (xs, s, m, l or xl), which
// ParseSize reads back. A Size outside the five constants returns a
// string naming its number, such as "Size(9)".
func (s Size) String() string {
	switch s {
	case SizeXS:
		return "xs"
	case SizeS:
		return "s"
	case SizeM:
		return "m"
	case SizeL:
		return "l"
	case SizeXL:
		return "xl"
	}
	return "Size(" + strconv.Itoa(int(s)) + ")"
}

// randBytes is the random field's width in bytes for the size and time
// form, and false for a Size outside the five constants.
func (s Size) randBytes(compact bool) (int, bool) {
	switch s {
	case SizeXS:
		if compact {
			return 1, true
		}
		return 0, true
	case SizeS:
		if compact {
			return 3, true
		}
		return 2, true
	case SizeM:
		if compact {
			return 6, true
		}
		return 5, true
	case SizeL:
		if compact {
			return 8, true
		}
		return 7, true
	case SizeXL:
		return 10, true
	}
	return 0, false
}

// Case is the letter case of an id's body. Case never applies to the
// prefix, and an upper-case body and its lower-case spelling are the same
// id. The zero value is CaseLower. A Case outside the two constants is
// refused with ErrUnknownCase.
type Case int

// The two cases.
const (
	// CaseLower is lower case, the default.
	CaseLower Case = iota
	// CaseUpper is upper case.
	CaseUpper
)

// Options are the inputs of Generate. The zero value asks for the defaults:
// no prefix, size xl, the size's random width, standard time, lower case,
// the delimiter "_", the current time and random bytes from
// crypto/rand.Reader.
type Options struct {
	// Prefix is written before the delimiter exactly as given; case never
	// applies to it, and it may contain the delimiter. The empty string,
	// the zero value, is no prefix: the id is the bare body, with no
	// delimiter.
	Prefix string

	// Size fixes the random field's width. The zero value is SizeXL.
	Size Size

	// RandSize is an explicit random field width in bytes, from 1 to 10,
	// in place of the size's width. The zero value takes the size's
	// width. Above 10 is ErrRandSizeTooLarge and below 0 is
	// ErrNegativeRandSize. A body of a width the size table does not list
	// has the same layout, but its length is one a decoder refuses.
	RandSize int

	// CompactTime asks for compact time: the low 40 bits of the time in 8
	// characters instead of 48 bits in 10, the freed byte handed to the
	// random field (except at SizeXL). A time whose first compact
	// character would be the deterministic marker, from mid-2038 until
	// the 40-bit window rolls over in 2039, is ErrCompactTimeOutOfRange.
	// The zero value, false, is standard time.
	CompactTime bool

	// Case is the letter case of the body. The zero value is CaseLower.
	Case Case

	// Delimiter is the string between the prefix and the body, written
	// only when there is a prefix. The empty string, the zero value, is
	// the delimiter "_". A delimiter made of Crockford Base32 characters
	// is not supported: a decoder could not tell where the body starts.
	Delimiter string

	// Time is the id's time, read to the millisecond (any finer part is
	// dropped), from the Unix epoch to 2^48 - 1 milliseconds after it;
	// outside that range is ErrTimeOutOfRange. The zero time.Time is the
	// current time from the system clock. The Unix epoch itself,
	// time.UnixMilli(0), is a time like any other, not the zero value.
	Time time.Time

	// Rand fixes the random field's bytes; a non-nil Rand, an empty one
	// included, must hold exactly the request's random width (any other
	// count is ErrRandWidthMismatch), and Reader is then not read. It
	// exists for reproducible ids, such as test vectors, and is never a
	// source of randomness: an id built from fixed bytes is exactly as
	// guessable as those bytes. Leave it nil for real ids.
	Rand []byte

	// Reader is where the random bytes come from when Rand is nil. Nil,
	// the zero value, is crypto/rand.Reader. Generate reads exactly the
	// random width from it with io.ReadFull, and does not read it when the
	// width is 0 or Rand is set. A Reader for real ids must be cryptographically
	// secure: the random field is what makes an id hard to guess. Any
	// other Reader belongs in tests only.
	Reader io.Reader
}
