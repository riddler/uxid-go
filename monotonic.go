package uxid

import (
	"crypto/rand"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// Draws is where a monotonic generator takes its random values from: a
// fresh random field, the seed, and the step that moves the field on
// within a millisecond. A nil Draws, wherever Monotonic takes one, is the
// default: draws read from the generator's byte source (Options.Reader,
// or crypto/rand.Reader), as Monotonic describes.
//
// A Draws of your own exists to replay known values, as a test or a
// conformance case does, and is never a source of randomness for real ids:
// a monotonic id is exactly as hard to guess as its draws.
type Draws interface {
	// Seed fills field, the random width of the generator in bytes, with a
	// fresh random value, read as one big-endian unsigned integer.
	Seed(field []byte) error

	// Step returns a step drawn uniformly from 1 to 2^(4 x width)
	// inclusive, where width is the random width in bytes. A step outside
	// that range is an error from the generating call.
	Step(width int) (uint64, error)
}

// Monotonic is a monotonic generator: the ids it makes increase strictly,
// in byte order as in time order, also when several fall in one
// millisecond and when the clock moves back. It follows the "Monotonic
// generation" rule of uxid-ex's wire-format guide. A monotonic id is an
// ordinary time-based id of its size and length: nothing in it says how
// it was made, and Decode reads it as any other.
//
// A generator is built for one prefix and one random width, from the
// Options NewMonotonic takes, and keeps the last time T it gave and the
// last random value R (the random field read as one big-endian unsigned
// integer of w bytes). For a call at time now:
//
//  1. with no last id, or now later than T, the id carries now and a fresh
//     random value, the seed: w bytes from the byte source;
//  2. with now equal to or earlier than T, the id carries T and R + step,
//     where step is drawn uniformly from 1 to 2^(4w) inclusive (half the
//     field's bit width): the low 4w bits of ceil(w / 2) bytes from the
//     byte source, read big-endian, plus one. The range holds a power of
//     two values, so the masked bits are exactly uniform over it, with no
//     modulo bias and nothing redrawn;
//  3. when R + step is 2^(8w) or more, the id carries T + 1 and a fresh
//     seed instead.
//
// The draws described are the default ones, from the byte source; a Draws
// passed to NextWith gives the seed and the step instead.
//
// A compact time holds only the low 40 bits of the time, so compact ids
// keep that order only within one 40-bit window, as every compact id does.
//
// The step is not a plus-one counter: consecutive ids in a millisecond are
// spread over a window of 2^(4w) values. That stops trivial enumeration but
// is a mitigation, not unpredictability, and a small random field stays
// small: a standard s id has 16 random bits.
//
// A Monotonic is safe for concurrent use by multiple goroutines: a
// sync.Mutex inside it orders the calls, so ids from one generator increase
// strictly whichever goroutine asks. The state lives in the value alone,
// so two generators never affect each other, and ids from two generators
// carry no order between them within a millisecond. Hold it by the pointer
// NewMonotonic returns; a copy of a Monotonic is not a generator.
type Monotonic struct {
	mu sync.Mutex

	prefix    string
	delimiter string
	alphabet  string
	compact   bool
	width     int
	fixed     time.Time
	reader    io.Reader

	started  bool
	lastTime uint64
	last     [maxRandBytes]byte
}

// NewMonotonic builds a monotonic generator from opts: its Prefix, Size,
// RandSize, CompactTime, Case, Delimiter, Time and Reader, each read as
// Generate reads it, with the same defaults. A non-zero Time is the time
// of every call that does not name one; ids then still increase, stepping
// within that millisecond and moving past it on overflow. The Reader is
// the byte source of the default draws.
//
// One default differs: at SizeXS the generator always uses compact time,
// whatever CompactTime says, because a standard xs has no random field to
// step and a compact xs has a one-byte field. Options cannot tell an unset
// CompactTime from false, so the wire format's refusal of an explicit
// standard-time monotonic xs has no request to answer here; RandSize, when
// set, still gives the random width.
//
// The refusals, each returned unwrapped and matched with errors.Is, in the
// order they are checked: ErrFromWithMonotonic for From, as a
// deterministic id has no random field to step; ErrRandWithMonotonic for
// fixed Rand bytes, as the generator chooses the random field;
// ErrUnknownSize and ErrUnknownCase; ErrNegativeRandSize and
// ErrRandSizeTooLarge for an explicit random width below 0 or above 10
// bytes; ErrEmptyRandWithMonotonic for a random width of 0 bytes
// (RandSize new(0)); and ErrTimeOutOfRange for a Time outside the time
// field. Both monotonic refusals are taken here rather than at each call,
// so a generator that exists can make ids.
func NewMonotonic(opts Options) (*Monotonic, error) {
	if opts.From != nil {
		return nil, ErrFromWithMonotonic
	}
	if opts.Rand != nil {
		return nil, ErrRandWithMonotonic
	}
	compact := opts.CompactTime || opts.Size == SizeXS
	width, ok := opts.Size.randBytes(compact)
	if !ok {
		return nil, ErrUnknownSize
	}
	if opts.Case != CaseLower && opts.Case != CaseUpper {
		return nil, ErrUnknownCase
	}
	if opts.RandSize != nil {
		switch {
		case *opts.RandSize < 0:
			return nil, ErrNegativeRandSize
		case *opts.RandSize > maxRandBytes:
			return nil, ErrRandSizeTooLarge
		}
		width = *opts.RandSize
	}
	if width == 0 {
		return nil, ErrEmptyRandWithMonotonic
	}
	if !opts.Time.IsZero() && !inTimeRange(opts.Time) {
		return nil, ErrTimeOutOfRange
	}
	alphabet := lowerAlphabet
	if opts.Case == CaseUpper {
		alphabet = upperAlphabet
	}
	delimiter := opts.Delimiter
	if delimiter == "" {
		delimiter = defaultDelimiter
	}
	reader := opts.Reader
	if reader == nil {
		reader = rand.Reader
	}
	return &Monotonic{
		prefix:    opts.Prefix,
		delimiter: delimiter,
		alphabet:  alphabet,
		compact:   compact,
		width:     width,
		fixed:     opts.Time,
		reader:    reader,
	}, nil
}

// Next returns the next id, with the default draws and the time from the
// Options NewMonotonic took (the system clock when that Time is zero). It
// is NextWith(nil, time.Time{}); see NextWith for the refusals.
func (m *Monotonic) Next() (string, error) {
	return m.NextWith(nil, time.Time{})
}

// NextWith returns the next id, taking its random values from draws and
// its time from now. A nil draws is the default draws from the byte source
// (see Monotonic). The zero now is the Time of the Options NewMonotonic
// took when that is set, and the system clock when it is not; any other now is the call's time, read
// to the millisecond. The id carries now, the last id's time, or the
// millisecond after it, as the rule on Monotonic says.
//
// NextWith returns ErrTimeOutOfRange for a time before the Unix epoch or
// above 2^48 - 1 ms, or for a step that overflows the field in millisecond
// 2^48 - 1, the last a time field holds; and ErrCompactTimeOutOfRange for
// a compact time the id would carry whose first character would be the
// deterministic marker, also when that time is the millisecond after the
// last id's, reached by an overflow. A failed draw returns an error that wraps the
// error of draws (for the default draws, the byte source's error, as
// Generate wraps it), and a step outside its range returns an error of
// its own; neither is a sentinel error.
//
// A call that returns an error leaves the generator as it was: the last
// id's time and random value stay what they were, and the next call
// starts from them. So after an overflow onto a refused compact time, a
// later call at or before the last id's time draws a new step: it is
// refused again if that step also overflows the field, and with a smaller
// step that does not, it returns an id in the last id's millisecond.
func (m *Monotonic) NextWith(draws Draws, now time.Time) (string, error) {
	if draws == nil {
		draws = readerDraws{reader: m.reader}
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	if now.IsZero() {
		now = m.fixed
	}
	if now.IsZero() {
		now = time.Now()
	}
	if !inTimeRange(now) {
		return "", ErrTimeOutOfRange
	}
	millis := uint64(now.UnixMilli())

	var value [maxRandBytes]byte
	field := value[:m.width]
	at := millis
	if m.started && millis <= m.lastTime {
		step, err := draws.Step(m.width)
		if err != nil {
			return "", fmt.Errorf("uxid: drawing a monotonic step: %w", err)
		}
		if step < 1 || step > 1<<(4*m.width) {
			return "", fmt.Errorf("uxid: monotonic step %d is outside 1 to 2^%d", step, 4*m.width)
		}
		copy(field, m.last[:m.width])
		at = m.lastTime
		if addStep(field, step) {
			if at == maxTime {
				return "", ErrTimeOutOfRange
			}
			at++
			if err := draws.Seed(field); err != nil {
				return "", fmt.Errorf("uxid: drawing a monotonic seed: %w", err)
			}
		}
	} else if err := draws.Seed(field); err != nil {
		return "", fmt.Errorf("uxid: drawing a monotonic seed: %w", err)
	}

	compactLow := at % compactWindow
	if m.compact && compactLow>>35 == marker {
		return "", ErrCompactTimeOutOfRange
	}

	var id strings.Builder
	id.Grow(len(m.prefix) + len(m.delimiter) + maxBodyLength)
	writePrefix(&id, m.prefix, m.delimiter)
	if m.compact {
		writeInteger(&id, compactLow, 8, m.alphabet)
	} else {
		writeInteger(&id, at, 10, m.alphabet)
	}
	writeBytes(&id, field, m.alphabet)

	m.started = true
	m.lastTime = at
	m.last = value
	return id.String(), nil
}

// inTimeRange reports whether at is within the time field: from the Unix
// epoch to 2^48 - 1 milliseconds after it.
func inTimeRange(at time.Time) bool {
	return !at.Before(time.UnixMilli(0)) && at.Before(time.UnixMilli(maxTime+1))
}

// addStep adds step to field, a big-endian unsigned integer, in place, and
// reports whether the sum overflowed the field's width.
func addStep(field []byte, step uint64) bool {
	carry := step
	for i := len(field) - 1; i >= 0 && carry > 0; i-- {
		sum := uint64(field[i]) + carry&0xff
		field[i] = byte(sum)
		carry = carry>>8 + sum>>8
	}
	return carry > 0
}

// readerDraws are the default draws: a seed is width bytes from reader,
// and a step the low 4 x width bits of ceil(width / 2) bytes from reader,
// read big-endian, plus one.
type readerDraws struct {
	reader io.Reader
}

// Seed fills field from the reader with io.ReadFull.
func (d readerDraws) Seed(field []byte) error {
	if _, err := io.ReadFull(d.reader, field); err != nil {
		return fmt.Errorf("reading %d random bytes: %w", len(field), err)
	}
	return nil
}

// Step reads ceil(width / 2) bytes, keeps their low 4 x width bits and adds
// one.
func (d readerDraws) Step(width int) (uint64, error) {
	var buf [maxRandBytes / 2]byte
	drawn := buf[:(width+1)/2]
	if _, err := io.ReadFull(d.reader, drawn); err != nil {
		return 0, fmt.Errorf("reading %d random bytes: %w", len(drawn), err)
	}
	var value uint64
	for _, b := range drawn {
		value = value<<8 | uint64(b)
	}
	return value&(1<<(4*width)-1) + 1, nil
}
