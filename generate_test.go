package uxid

import (
	"bytes"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

// corpusTime is the time most corpus encode cases use, 1700000000000 ms.
var corpusTime = time.UnixMilli(1_700_000_000_000)

// unhex decodes a corpus rand field. An empty string decodes to an empty,
// non-nil slice: in the corpus an empty rand is a fixed field of no bytes.
func unhex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("hex %q: %v", s, err)
	}
	if b == nil {
		b = []byte{}
	}
	return b
}

// failingReader fails the test when Generate reads from it.
type failingReader struct{ t *testing.T }

func (r failingReader) Read([]byte) (int, error) {
	r.t.Fatal("Generate read from the Reader")
	return 0, nil
}

// errReader returns its error on every read.
type errReader struct{ err error }

func (r errReader) Read([]byte) (int, error) { return 0, r.err }

// Hand-picked cases from the conformance corpus's encode.json (uxid-ex
// v2.10.0), by their case ids, until the corpus runner vendors the file:
// every size in both time forms, both cases, the partial-group rule, the
// time range's ends, the compact guard's last valid time, no prefix, an
// empty prefix and a prefix holding the delimiter.
//
// sabotage: in writeBytes, write the short last group shifted left
// (alphabet[buffer<<(5-bits)]); the all-ones and every odd-width case go red.
func TestGenerateCorpusEncodeCases(t *testing.T) {
	cases := []struct {
		id       string
		prefix   string
		size     Size
		compact  bool
		upper    bool
		millis   int64
		rand     string
		expected string
	}{
		{"encode/xs-standard-lower", "usr", SizeXS, false, false, 1_700_000_000_000, "", "usr_01hf7yat00"},
		{"encode/xs-standard-upper", "usr", SizeXS, false, true, 1_700_000_000_000, "", "usr_01HF7YAT00"},
		{"encode/xs-compact-upper", "usr", SizeXS, true, true, 1_700_000_000_000, "01", "usr_HF7YAT0001"},
		{"encode/s-standard-lower", "usr", SizeS, false, false, 1_700_000_000_000, "0123", "usr_01hf7yat0004h1"},
		{"encode/s-compact-lower", "usr", SizeS, true, false, 1_700_000_000_000, "012345", "usr_hf7yat0004hm5"},
		{"encode/m-standard-upper", "usr", SizeM, false, true, 1_700_000_000_000, "0123456789", "usr_01HF7YAT0004HMASW9"},
		{"encode/m-compact-lower", "usr", SizeM, true, false, 1_700_000_000_000, "0123456789ab", "usr_hf7yat0004hmasw9n3"},
		{"encode/l-standard-lower", "usr", SizeL, false, false, 1_700_000_000_000, "0123456789abcd", "usr_01hf7yat0004hmasw9nf61"},
		{"encode/l-compact-upper", "usr", SizeL, true, true, 1_700_000_000_000, "0123456789abcdef", "usr_HF7YAT0004HMASW9NF6YF"},
		{"encode/xl-standard-lower", "usr", SizeXL, false, false, 1_700_000_000_000, "0123456789abcdef0123", "usr_01hf7yat0004hmasw9nf6yy093"},
		{"encode/xl-compact-lower", "usr", SizeXL, true, false, 1_700_000_000_000, "0123456789abcdef0123", "usr_hf7yat0004hmasw9nf6yy093"},
		{"encode/xs-compact-rand-all-ones", "cus", SizeXS, true, false, 1_700_000_000_000, "ff", "cus_hf7yat00z7"},
		{"encode/s-standard-rand-all-ones", "cus", SizeS, false, false, 1_700_000_000_000, "ffff", "cus_01hf7yat00zzz1"},
		{"encode/m-compact-rand-all-ones", "cus", SizeM, true, false, 1_700_000_000_000, "ffffffffffff", "cus_hf7yat00zzzzzzzzz7"},
		{"encode/l-compact-rand-all-ones", "cus", SizeL, true, false, 1_700_000_000_000, "ffffffffffffffff", "cus_hf7yat00zzzzzzzzzzzzf"},
		{"encode/xl-standard-rand-all-ones", "cus", SizeXL, false, false, 1_700_000_000_000, "ffffffffffffffffffff", "cus_01hf7yat00zzzzzzzzzzzzzzzz"},
		{"encode/xs-standard-time-zero", "", SizeXS, false, false, 0, "", "0000000000"},
		{"encode/xs-standard-time-max", "", SizeXS, false, false, 281_474_976_710_655, "", "7zzzzzzzzz"},
		{"encode/s-compact-guard-last-valid", "usr", SizeS, true, false, 1_065_151_889_407, "012345", "usr_yzzzzzzz04hm5"},
		{"encode/xl-standard-no-prefix", "", SizeXL, false, false, 1_700_000_000_000, "0123456789abcdef0123", "01hf7yat0004hmasw9nf6yy093"},
		{"encode/xl-standard-empty-prefix", "", SizeXL, false, false, 1_700_000_000_000, "0123456789abcdef0123", "01hf7yat0004hmasw9nf6yy093"},
		{"encode/s-standard-prefix-with-delimiter", "a_b", SizeS, false, false, 1_700_000_000_000, "0123", "a_b_01hf7yat0004h1"},
	}
	for _, c := range cases {
		t.Run(c.id, func(t *testing.T) {
			opts := Options{
				Prefix:      c.prefix,
				Size:        c.size,
				CompactTime: c.compact,
				Time:        time.UnixMilli(c.millis),
				Rand:        unhex(t, c.rand),
				Reader:      failingReader{t},
			}
			if c.upper {
				opts.Case = CaseUpper
			}
			got, err := Generate(opts)
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			if got != c.expected {
				t.Fatalf("Generate = %q, want %q", got, c.expected)
			}
		})
	}
}

// Hand-picked generate cases from the corpus's invalid.json (uxid-ex
// v2.10.0): every generation error a time-based request can meet.
//
// sabotage: in Generate, compare compactLow>>35 against 30 instead of the
// marker; both guard cases go red.
func TestGenerateCorpusRefusals(t *testing.T) {
	cases := []struct {
		id       string
		opts     Options
		expected error
	}{
		{"invalid/generate-compact-time-at-guard",
			Options{Prefix: "usr", Size: SizeS, CompactTime: true, Time: time.UnixMilli(2_164_663_517_184), Rand: []byte{1, 2, 3}},
			ErrCompactTimeOutOfRange},
		{"invalid/generate-compact-time-at-guard-previous-window",
			Options{Prefix: "usr", Size: SizeS, CompactTime: true, Time: time.UnixMilli(1_065_151_889_408), Rand: []byte{1, 2, 3}},
			ErrCompactTimeOutOfRange},
		{"invalid/generate-rand-too-wide",
			Options{Prefix: "usr", Size: SizeS, Time: corpusTime, Rand: []byte{1, 2, 3}},
			ErrRandWidthMismatch},
		{"invalid/generate-rand-too-narrow",
			Options{Prefix: "usr", Size: SizeXL, Time: corpusTime, Rand: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}},
			ErrRandWidthMismatch},
		{"invalid/generate-rand-on-xs-standard",
			Options{Prefix: "usr", Size: SizeXS, Time: corpusTime, Rand: []byte{1}},
			ErrRandWidthMismatch},
		{"invalid/generate-rand-size-11",
			Options{Prefix: "usr", RandSize: 11, Size: SizeXL, Time: corpusTime},
			ErrRandSizeTooLarge},
	}
	for _, c := range cases {
		t.Run(c.id, func(t *testing.T) {
			c.opts.Reader = failingReader{t}
			got, err := Generate(c.opts)
			if !errors.Is(err, c.expected) {
				t.Fatalf("Generate = %q, %v; want error %v", got, err, c.expected)
			}
			if got != "" {
				t.Fatalf("Generate returned %q with its error", got)
			}
			if err.Error() != c.expected.Error() {
				t.Fatalf("error text %q, want the corpus name %q", err.Error(), c.expected.Error())
			}
		})
	}
}

// sabotage: change ErrRandWithMonotonic's text to "rand_with_monotonic_x";
// this test goes red.
func TestSentinelTextsAreTheCorpusNames(t *testing.T) {
	sentinels := map[error]string{
		ErrCompactTimeOutOfRange: "compact_time_out_of_range",
		ErrRandWidthMismatch:     "rand_width_mismatch",
		ErrRandSizeTooLarge:      "rand_size_too_large",
		ErrFromWithMonotonic:     "from_with_monotonic",
		ErrRandWithFrom:          "rand_with_from",
		ErrRandWithMonotonic:     "rand_with_monotonic",
		ErrTimeOutOfRange:        "time_out_of_range",
		ErrNegativeRandSize:      "negative_rand_size",
		ErrUnknownSize:           "unknown_size",
		ErrUnknownCase:           "unknown_case",
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
	got, err := Generate(Options{Prefix: "cus", RandSize: 1, Time: corpusTime, Rand: []byte{0xff}})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if want := "cus_01hf7yat00z7"; got != want {
		t.Fatalf("Generate = %q, want %q", got, want)
	}

	// An explicit width overrides the size's, and the Reader gives exactly
	// that many bytes.
	reader := bytes.NewReader(bytes.Repeat([]byte{0xff}, 12))
	got, err = Generate(Options{Prefix: "cus", Size: SizeS, RandSize: 10, Time: corpusTime, Reader: reader})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if want := "cus_01hf7yat00zzzzzzzzzzzzzzzz"; got != want {
		t.Fatalf("Generate = %q, want %q", got, want)
	}
	if reader.Len() != 2 {
		t.Fatalf("Generate read %d bytes, want 10", 12-reader.Len())
	}

	if _, err := Generate(Options{RandSize: -1, Time: corpusTime}); !errors.Is(err, ErrNegativeRandSize) {
		t.Fatalf("RandSize -1: %v, want ErrNegativeRandSize", err)
	}
	if _, err := Generate(Options{RandSize: 11, Rand: make([]byte, 11), Time: corpusTime}); !errors.Is(err, ErrRandSizeTooLarge) {
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
