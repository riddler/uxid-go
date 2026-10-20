package uxid

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

// corpusTime is the time most corpus encode cases use, 1700000000000 ms.
var corpusTime = time.UnixMilli(1_700_000_000_000)

// failingReader fails the test when Generate reads from it.
type failingReader struct{ t *testing.T }

func (r failingReader) Read([]byte) (int, error) {
	r.t.Fatal("Generate read from the Reader")
	return 0, nil
}

// errReader returns its error on every read.
type errReader struct{ err error }

func (r errReader) Read([]byte) (int, error) { return 0, r.err }

// sabotage: change ErrRandWithMonotonic's text to "rand_with_monotonic_x";
// this test goes red.
func TestSentinelTextsAreTheCorpusNames(t *testing.T) {
	sentinels := map[error]string{
		ErrCompactTimeOutOfRange:  "compact_time_out_of_range",
		ErrRandWidthMismatch:      "rand_width_mismatch",
		ErrRandSizeTooLarge:       "rand_size_too_large",
		ErrFromWithMonotonic:      "from_with_monotonic",
		ErrRandWithFrom:           "rand_with_from",
		ErrRandWithMonotonic:      "rand_with_monotonic",
		ErrTimeOutOfRange:         "time_out_of_range",
		ErrNegativeRandSize:       "negative_rand_size",
		ErrUnknownSize:            "unknown_size",
		ErrUnknownCase:            "unknown_case",
		ErrEmptyRandWithMonotonic: "empty_rand_with_monotonic",
	}
	for err, name := range sentinels {
		if err.Error() != name {
			t.Errorf("error text %q, want %q", err.Error(), name)
		}
		for other := range sentinels {
			if other != err && errors.Is(err, other) {
				t.Errorf("%v matches %v", err, other)
			}
		}
	}
}

// sabotage: in Size.randBytes, return 10 for every size; the default is
// unchanged but the explicit SizeS options go red.
func TestGenerateZeroOptionsAreTheDefaults(t *testing.T) {
	random := bytes.Repeat([]byte{0}, 10)
	got, err := Generate(Options{Time: corpusTime, Reader: bytes.NewReader(random)})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	// No prefix, size xl (26 characters), standard time, lower case.
	if want := "01hf7yat000000000000000000"; got != want {
		t.Fatalf("Generate = %q, want %q", got, want)
	}

	explicit, err := Generate(Options{
		Prefix: "cus", Size: SizeS, Case: CaseLower, Delimiter: "_",
		Time: corpusTime, Reader: bytes.NewReader([]byte{0xff, 0xff}),
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	implicit, err := Generate(Options{
		Prefix: "cus", Size: SizeS,
		Time: corpusTime, Reader: bytes.NewReader([]byte{0xff, 0xff}),
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if explicit != "cus_01hf7yat00zzz1" || implicit != explicit {
		t.Fatalf("explicit %q, implicit %q; want both cus_01hf7yat00zzz1", explicit, implicit)
	}
}

// sabotage: in Generate, write the delimiter even when the prefix is empty;
// the no-prefix custom-delimiter row goes red.
func TestGenerateDelimiter(t *testing.T) {
	cases := []struct {
		prefix, delimiter, expected string
	}{
		{"usr", "-", "usr-01hf7yat00"},
		{"usr", "::", "usr::01hf7yat00"},
		{"usr", "", "usr_01hf7yat00"},
		{"", "-", "01hf7yat00"},
	}
	for _, c := range cases {
		got, err := Generate(Options{Prefix: c.prefix, Delimiter: c.delimiter, Size: SizeXS, Time: corpusTime})
		if err != nil {
			t.Fatalf("Generate(%q, %q): %v", c.prefix, c.delimiter, err)
		}
		if got != c.expected {
			t.Errorf("Generate(%q, %q) = %q, want %q", c.prefix, c.delimiter, got, c.expected)
		}
	}
}

// sabotage: in Generate, upper-case the prefix along with the body
// (strings.ToUpper(opts.Prefix)); this test goes red.
func TestGenerateCaseNeverAppliesToThePrefix(t *testing.T) {
	got, err := Generate(Options{Prefix: "Usr", Case: CaseUpper, Size: SizeS, Time: corpusTime, Rand: []byte{0x01, 0x23}})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if want := "Usr_01HF7YAT0004H1"; got != want {
		t.Fatalf("Generate = %q, want %q", got, want)
	}
}

// sabotage: in Generate, ignore opts.RandSize (drop the case that sets
// width from it); the explicit widths go red.
func TestGenerateRandSize(t *testing.T) {
	got, err := Generate(Options{Prefix: "cus", RandSize: new(1), Time: corpusTime, Rand: []byte{0xff}})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if want := "cus_01hf7yat00z7"; got != want {
		t.Fatalf("Generate = %q, want %q", got, want)
	}

	// An explicit width overrides the size's, and the Reader gives exactly
	// that many bytes.
	reader := bytes.NewReader(bytes.Repeat([]byte{0xff}, 12))
	got, err = Generate(Options{Prefix: "cus", Size: SizeS, RandSize: new(10), Time: corpusTime, Reader: reader})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if want := "cus_01hf7yat00zzzzzzzzzzzzzzzz"; got != want {
		t.Fatalf("Generate = %q, want %q", got, want)
	}
	if reader.Len() != 2 {
		t.Fatalf("Generate read %d bytes, want 10", 12-reader.Len())
	}

	if _, err := Generate(Options{RandSize: new(-1), Time: corpusTime}); !errors.Is(err, ErrNegativeRandSize) {
		t.Fatalf("RandSize -1: %v, want ErrNegativeRandSize", err)
	}
	if _, err := Generate(Options{RandSize: new(11), Rand: make([]byte, 11), Time: corpusTime}); !errors.Is(err, ErrRandSizeTooLarge) {
		t.Fatalf("RandSize 11 with 11 fixed bytes: %v, want ErrRandSizeTooLarge", err)
	}
}

// sabotage: in Generate, compare the time against maxTime+2 instead of
// maxTime+1; the first out-of-range time goes red.
func TestGenerateTimeRange(t *testing.T) {
	refused := []time.Time{
		time.UnixMilli(1 << 48),
		time.UnixMilli(-1),
		time.Unix(0, -1),
		time.Date(1969, time.December, 31, 23, 59, 59, 0, time.UTC),
		time.Date(1_000_000, time.January, 1, 0, 0, 0, 0, time.UTC),
	}
	for _, at := range refused {
		got, err := Generate(Options{Size: SizeXS, Time: at})
		if !errors.Is(err, ErrTimeOutOfRange) {
			t.Errorf("Generate at %v = %q, %v; want ErrTimeOutOfRange", at, got, err)
		}
	}

	// The time is read to the millisecond: the last nanosecond of the last
	// millisecond is still in range, and the epoch is not the zero value.
	accepted := map[string]time.Time{
		"7zzzzzzzzz": time.UnixMilli(1<<48 - 1).Add(999_999 * time.Nanosecond),
		"0000000000": time.UnixMilli(0),
		"0000000001": time.Unix(0, 1_999_999),
	}
	for want, at := range accepted {
		got, err := Generate(Options{Size: SizeXS, Time: at})
		if err != nil || got != want {
			t.Errorf("Generate at %v = %q, %v; want %q", at, got, err, want)
		}
	}
}

// sabotage: in Generate, drop the check that opts.Case is one of the two
// constants; the Case(2) row goes red.
func TestGenerateUnknownSizeAndCase(t *testing.T) {
	for _, size := range []Size{Size(-1), Size(5), Size(99)} {
		if _, err := Generate(Options{Size: size, Time: corpusTime}); !errors.Is(err, ErrUnknownSize) {
			t.Errorf("Size(%d): %v, want ErrUnknownSize", int(size), err)
		}
	}
	for _, c := range []Case{Case(-1), Case(2)} {
		if _, err := Generate(Options{Case: c, Time: corpusTime}); !errors.Is(err, ErrUnknownCase) {
			t.Errorf("Case(%d): %v, want ErrUnknownCase", int(c), err)
		}
	}
}

// sabotage: in Generate, set random to nil before the Rand check, so fixed
// bytes are drawn from the Reader instead; the fixed Rand row goes red.
func TestGenerateReadsTheReaderOnlyWhenItMust(t *testing.T) {
	cases := map[string]Options{
		"xs standard, width 0":   {Size: SizeXS, Time: corpusTime},
		"fixed Rand":             {Size: SizeS, Time: corpusTime, Rand: []byte{1, 2}},
		"fixed empty Rand on xs": {Size: SizeXS, Time: corpusTime, Rand: []byte{}},
	}
	for name, opts := range cases {
		opts.Reader = failingReader{t}
		if _, err := Generate(opts); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}

	// A non-nil empty Rand is a fixed field of no bytes, not an unset one.
	if _, err := Generate(Options{Size: SizeS, Time: corpusTime, Rand: []byte{}, Reader: failingReader{t}}); !errors.Is(err, ErrRandWidthMismatch) {
		t.Errorf("empty Rand on s: %v, want ErrRandWidthMismatch", err)
	}
}

// sabotage: in Generate, return a bare ErrRandWidthMismatch in place of the
// wrapped Reader error; both rows go red.
func TestGenerateWrapsAReaderFailure(t *testing.T) {
	broken := errors.New("source unavailable")
	_, err := Generate(Options{Time: corpusTime, Reader: errReader{broken}})
	if !errors.Is(err, broken) {
		t.Fatalf("Generate: %v, want an error wrapping %v", err, broken)
	}

	// A Reader that ends early is an unexpected EOF, not a short id.
	got, err := Generate(Options{Size: SizeS, Time: corpusTime, Reader: bytes.NewReader([]byte{1})})
	if !errors.Is(err, io.ErrUnexpectedEOF) || got != "" {
		t.Fatalf("Generate = %q, %v; want io.ErrUnexpectedEOF", got, err)
	}
	for _, sentinel := range []error{ErrRandWidthMismatch, ErrRandSizeTooLarge, ErrCompactTimeOutOfRange, ErrTimeOutOfRange} {
		if errors.Is(err, sentinel) {
			t.Fatalf("a Reader failure matches %v", sentinel)
		}
	}
}

// The defaults read the system clock and crypto/rand.Reader. The clock is
// read once around the call and the id's time is only checked to fall
// within an hour of it: two clock reads are never compared for order.
//
// sabotage: in Generate, use time.UnixMilli(0) in place of time.Now() for
// the zero Time; the time check goes red.
func TestGenerateDefaultClockAndReader(t *testing.T) {
	got, err := Generate(Options{Prefix: "usr"})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	body, ok := strings.CutPrefix(got, "usr_")
	if !ok || len(body) != 26 {
		t.Fatalf("Generate = %q, want usr_ and a 26-character body", got)
	}
	for _, r := range body {
		if !strings.ContainsRune(lowerAlphabet, r) {
			t.Fatalf("body %q holds %q, outside the lower-case alphabet", body, r)
		}
	}
	millis := readStandardTime(t, body[:10])
	if drift := time.Since(time.UnixMilli(millis)).Abs(); drift > time.Hour {
		t.Fatalf("id time %d ms is %v from the clock", millis, drift)
	}
}

// readStandardTime reads a 10-character lower-case standard time field.
func readStandardTime(t *testing.T, field string) int64 {
	t.Helper()
	var millis int64
	for _, r := range field {
		v := strings.IndexRune(lowerAlphabet, r)
		if v < 0 {
			t.Fatalf("time field %q holds %q", field, r)
		}
		millis = millis<<5 | int64(v)
	}
	return millis
}

// Lengths are counted in bytes: a prefix and a delimiter outside ASCII add
// their byte lengths, and the body stays ASCII.
//
// sabotage: in Generate, cut the prefix to its length in runes
// (opts.Prefix[:len([]rune(opts.Prefix))]); this test goes red.
func TestGenerateLengthIsInBytes(t *testing.T) {
	prefix, delimiter := "rés", "·"
	got, err := Generate(Options{Prefix: prefix, Delimiter: delimiter, Size: SizeM, Time: corpusTime, Rand: make([]byte, 5)})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.HasPrefix(got, prefix+delimiter) {
		t.Fatalf("Generate = %q, want the prefix and delimiter first", got)
	}
	if want := len(prefix) + len(delimiter) + 18; len(got) != want {
		t.Fatalf("len(%q) = %d bytes, want %d", got, len(got), want)
	}
}

// A request with two faults is refused for the first in the reference's
// order: the random width (rand_size_too_large, then rand_width_mismatch)
// before the compact-time guard. The time is the corpus's guard case,
// invalid/generate-compact-time-at-guard.
//
// sabotage: in Generate, move the compact-time guard above the RandSize
// checks; both rows go red.
func TestGenerateRefusesTheWidthBeforeTheCompactGuard(t *testing.T) {
	guard := time.UnixMilli(2_164_663_517_184)
	cases := map[error]Options{
		ErrRandSizeTooLarge:  {Prefix: "usr", Size: SizeS, CompactTime: true, Time: guard, RandSize: new(11)},
		ErrRandWidthMismatch: {Prefix: "usr", Size: SizeS, CompactTime: true, Time: guard, Rand: []byte{1, 2}},
	}
	for want, opts := range cases {
		opts.Reader = failingReader{t}
		if _, err := Generate(opts); !errors.Is(err, want) {
			t.Errorf("Generate: %v, want %v", err, want)
		}
	}
}

// An explicit width of 0 is no random field, in either time form, and
// differs from an unset RandSize, which takes the size's width.
//
// sabotage: in Generate, apply RandSize only when *opts.RandSize > 0; both
// rows go red.
func TestGenerateExplicitRandSizeZero(t *testing.T) {
	cases := map[string]Options{
		"usr_01hf7yat00": {Prefix: "usr", Size: SizeXL, Time: corpusTime, RandSize: new(0)},
		"usr_hf7yat00":   {Prefix: "usr", Size: SizeXS, CompactTime: true, Time: corpusTime, RandSize: new(0), Rand: []byte{}},
	}
	for want, opts := range cases {
		opts.Reader = failingReader{t}
		got, err := Generate(opts)
		if err != nil || got != want {
			t.Errorf("Generate = %q, %v; want %q", got, err, want)
		}
	}
}

// A Reader that gives no byte at all fails with io.EOF, wrapped.
//
// sabotage: in Generate, format the Reader's error with %v in place of
// %w; this test goes red.
func TestGenerateReaderAtEOF(t *testing.T) {
	got, err := Generate(Options{Size: SizeS, Time: corpusTime, Reader: bytes.NewReader(nil)})
	if !errors.Is(err, io.EOF) || got != "" {
		t.Fatalf("Generate = %q, %v; want io.EOF", got, err)
	}
}
