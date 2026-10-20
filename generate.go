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

// Generate returns a time-based id built from opts: the prefix and the
// delimiter (when there is a prefix), then a body of the time field and
// the random field in Crockford Base32, as uxid-ex's wire-format guide lays
// it out. Each zero field of opts takes its default (see Options).
//
// Every body character is one ASCII byte, so an id's length in bytes is
// the prefix's length in bytes, plus the delimiter's when there is a
// prefix, plus the body length the size table lists.
//
// Generate reads the random field from opts.Reader (crypto/rand.Reader
// when it is nil) unless opts.Rand fixes it. A failed read returns an
// error that wraps the Reader's error, and an io.ErrUnexpectedEOF for a
// Reader that ends early; it is none of the sentinel errors.
//
// The refusals, each returned unwrapped and matched with errors.Is:
// ErrUnknownSize and ErrUnknownCase for a Size or Case outside its
// constants; ErrTimeOutOfRange for a time outside the time field;
// ErrCompactTimeOutOfRange for a compact time whose first character would
// be the deterministic marker; ErrNegativeRandSize and ErrRandSizeTooLarge
// for an explicit random width below 0 or above 10 bytes; and
// ErrRandWidthMismatch for fixed random bytes of another width.
func Generate(opts Options) (string, error) {
	width, ok := opts.Size.randBytes(opts.CompactTime)
	if !ok {
		return "", ErrUnknownSize
	}
	if opts.Case != CaseLower && opts.Case != CaseUpper {
		return "", ErrUnknownCase
	}

	at := opts.Time
	if at.IsZero() {
		at = time.Now()
	}
	if at.Before(time.UnixMilli(0)) || !at.Before(time.UnixMilli(maxTime+1)) {
		return "", ErrTimeOutOfRange
	}
	millis := uint64(at.UnixMilli())
	compactLow := millis % (1 << 40)
	if opts.CompactTime && compactLow>>35 == marker {
		return "", ErrCompactTimeOutOfRange
	}

	switch {
	case opts.RandSize < 0:
		return "", ErrNegativeRandSize
	case opts.RandSize > maxRandBytes:
		return "", ErrRandSizeTooLarge
	case opts.RandSize > 0:
		width = opts.RandSize
	}

	random := opts.Rand
	if random != nil {
		if len(random) != width {
			return "", ErrRandWidthMismatch
		}
	} else {
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

	delimiter := opts.Delimiter
	if delimiter == "" {
		delimiter = defaultDelimiter
	}
	alphabet := lowerAlphabet
	if opts.Case == CaseUpper {
		alphabet = upperAlphabet
	}

	var id strings.Builder
	id.Grow(len(opts.Prefix) + len(delimiter) + maxBodyLength)
	if opts.Prefix != "" {
		id.WriteString(opts.Prefix)
		id.WriteString(delimiter)
	}
	if opts.CompactTime {
		writeInteger(&id, compactLow, 8, alphabet)
	} else {
		writeInteger(&id, millis, 10, alphabet)
	}
	writeBytes(&id, random, alphabet)
	return id.String(), nil
}
