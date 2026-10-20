package uxid

import (
	"crypto/rand"
	"fmt"
	"io"
	"strings"
	"time"
)

const (
	// maxTime is the latest time a time field holds: 2^48 - 1 milliseconds.
	maxTime = 1<<48 - 1
	// maxRandBytes is the widest random field at any size, in bytes.
	maxRandBytes = 10
	// marker is the deterministic marker's value, which a compact time may
	// not start with.
	marker = 31
	// defaultDelimiter is the delimiter an empty Options.Delimiter stands for.
	defaultDelimiter = "_"
	// maxBodyLength is the longest body, a standard xl one, in bytes.
	maxBodyLength = 26
)

// Generate returns an id built from opts: the prefix and the delimiter
// (when there is a prefix), then a body in Crockford Base32, as uxid-ex's
// wire-format guide lays it out. Each zero field of opts takes its default
// (see Options).
//
// The body is time-based unless opts.From is set: the time field, then the
// random field. With opts.From set it is deterministic: the marker (z, or
// Z in upper case), then hash characters of SHA-256 over the prefix and
// *opts.From, so the same prefix, From and Size always give the same id.
//
// Every body character is one ASCII byte, so an id's length in bytes is
// the prefix's length in bytes, plus the delimiter's when there is a
// prefix, plus the body length. The body length is the size table's (see
// Size) for a deterministic body and for a time-based one at the size's
// random width; an explicit RandSize gives a time field of 10 characters
// (8 for compact time) plus one character for each 5 bits of the random
// field, rounded up.
//
// For a time-based id, Generate reads the random field from opts.Reader
// (crypto/rand.Reader when it is nil) unless opts.Rand fixes it. A failed
// read returns an error that wraps the Reader's error: io.EOF for a Reader
// that gives no byte, io.ErrUnexpectedEOF for one that ends early, or the
// error the Reader returned. It is none of the sentinel errors. A
// deterministic id reads neither the Reader nor the clock.
//
// The refusals, each returned unwrapped and matched with errors.Is:
// ErrUnknownSize and ErrUnknownCase for a Size or Case outside its
// constants; ErrRandWithFrom for fixed random bytes with From; and, for a
// time-based id, ErrTimeOutOfRange for a time outside the time field,
// ErrNegativeRandSize and ErrRandSizeTooLarge for an explicit random width
// below 0 or above 10 bytes, ErrRandWidthMismatch for fixed random bytes
// of another width, and ErrCompactTimeOutOfRange for a compact time whose
// first character would be the deterministic marker. A time-based request
// with more than one fault is refused for the first in that order, which
// is the reference's order for the corpus's names: the random width before
// the compact-time guard.
func Generate(opts Options) (string, error) {
	width, ok := opts.Size.randBytes(opts.CompactTime)
	if !ok {
		return "", ErrUnknownSize
	}
	if opts.Case != CaseLower && opts.Case != CaseUpper {
		return "", ErrUnknownCase
	}
	alphabet := lowerAlphabet
	if opts.Case == CaseUpper {
		alphabet = upperAlphabet
	}
	delimiter := opts.Delimiter
	if delimiter == "" {
		delimiter = defaultDelimiter
	}

	if opts.From != nil {
		count := opts.Size.hashChars()
		if opts.Rand != nil {
			return "", ErrRandWithFrom
		}
		var id strings.Builder
		id.Grow(len(opts.Prefix) + len(delimiter) + maxBodyLength)
		writePrefix(&id, opts.Prefix, delimiter)
		writeDeterministic(&id, opts.Prefix, *opts.From, count, alphabet)
		return id.String(), nil
	}

	at := opts.Time
	if at.IsZero() {
		at = time.Now()
	}
	if at.Before(time.UnixMilli(0)) || !at.Before(time.UnixMilli(maxTime+1)) {
		return "", ErrTimeOutOfRange
	}
	millis := uint64(at.UnixMilli())

	if opts.RandSize != nil {
		switch {
		case *opts.RandSize < 0:
			return "", ErrNegativeRandSize
		case *opts.RandSize > maxRandBytes:
			return "", ErrRandSizeTooLarge
		}
		width = *opts.RandSize
	}
	if opts.Rand != nil && len(opts.Rand) != width {
		return "", ErrRandWidthMismatch
	}

	compactLow := millis % (1 << 40)
	if opts.CompactTime && compactLow>>35 == marker {
		return "", ErrCompactTimeOutOfRange
	}

	random := opts.Rand
	if random == nil {
		random = make([]byte, width)
		if width > 0 {
			reader := opts.Reader
			if reader == nil {
				reader = rand.Reader
			}
			if _, err := io.ReadFull(reader, random); err != nil {
				return "", fmt.Errorf("uxid: reading %d random bytes: %w", width, err)
			}
		}
	}

	var id strings.Builder
	id.Grow(len(opts.Prefix) + len(delimiter) + maxBodyLength)
	writePrefix(&id, opts.Prefix, delimiter)
	if opts.CompactTime {
		writeInteger(&id, compactLow, 8, alphabet)
	} else {
		writeInteger(&id, millis, 10, alphabet)
	}
	writeBytes(&id, random, alphabet)
	return id.String(), nil
}

// writePrefix writes the prefix and the delimiter, or nothing for no
// prefix.
func writePrefix(out *strings.Builder, prefix, delimiter string) {
	if prefix != "" {
		out.WriteString(prefix)
		out.WriteString(delimiter)
	}
}
