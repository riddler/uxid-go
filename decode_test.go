package uxid

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// The decoder clock window the conformance corpus's README states (uxid-ex
// v2.10.0): every compact decode case holds for a clock from
// 1699913600000 ms (2023-11-13T22:13:20.000Z) inclusive to 2164577117183 ms
// (2038-08-04T23:25:17.183Z) exclusive. The start is one day before the
// cases at 1700000000000 ms; the end is 2^40 + 1065151889407 less one day,
// the last clock at which the compact guard's last valid time from the
// window before is still more than one day ahead.
const (
	windowStart int64 = 1_699_913_600_000
	windowEnd   int64 = 2_164_577_117_183
)

// decodeCase is a hand-picked decode case from the corpus's decode.json
// (uxid-ex v2.10.0), by its case id. A deterministic case has millis -1.
type decodeCase struct {
	id            string
	input         string
	prefix        string
	size          Size
	compact       bool
	deterministic bool
	millis        int64
}

// Every decode case of the corpus, until the corpus runner vendors the
// file: every size in both time forms, the 10 and 18 readings, upper case,
// no prefix, a prefix holding the delimiter, the compact guard's last valid
// time and the deterministic bodies.
var decodeCases = []decodeCase{
	{"decode/xl-standard", "usr_01hf7yat0004hmasw9nf6yy093", "usr", SizeXL, false, false, 1_700_000_000_000},
	{"decode/xl-compact", "usr_hf7yat0004hmasw9nf6yy093", "usr", SizeXL, true, false, 1_700_000_000_000},
	{"decode/l-standard", "usr_01hf7yat0004hmasw9nf61", "usr", SizeL, false, false, 1_700_000_000_000},
	{"decode/l-compact", "usr_hf7yat0004hmasw9nf6yf", "usr", SizeL, true, false, 1_700_000_000_000},
	{"decode/m-standard", "usr_01hf7yat0004hmasw9", "usr", SizeM, false, false, 1_700_000_000_000},
	{"decode/m-compact-reads-as-standard", "usr_hf7yat0004hmasw9n3", "usr", SizeM, false, false, 51_950_139_736_068},
	{"decode/s-standard", "usr_01hf7yat0004h1", "usr", SizeS, false, false, 1_700_000_000_000},
	{"decode/s-compact", "usr_hf7yat0004hm5", "usr", SizeS, true, false, 1_700_000_000_000},
	{"decode/xs-standard", "usr_01hf7yat00", "usr", SizeXS, false, false, 1_700_000_000_000},
	{"decode/xs-compact-reads-as-standard", "usr_hf7yat0001", "usr", SizeXS, false, false, 51_950_139_736_065},
	{"decode/xl-standard-upper", "usr_01HF7YAT0004HMASW9NF6YY093", "usr", SizeXL, false, false, 1_700_000_000_000},
	{"decode/xl-standard-no-prefix", "01hf7yat0004hmasw9nf6yy093", "", SizeXL, false, false, 1_700_000_000_000},
	{"decode/s-standard-prefix-with-delimiter", "a_b_01hf7yat000001", "a_b", SizeS, false, false, 1_700_000_000_000},
	{"decode/s-compact-guard-last-valid", "usr_yzzzzzzz04hm5", "usr", SizeS, true, false, 1_065_151_889_407},
	{"decode/deterministic-usr-xl", "usr_zcvt7epac0t1ebcsjfyf7cwz25", "usr", SizeXL, false, true, -1},
	{"decode/deterministic-cus-s", "cus_zt9vnyfd85bff0", "cus", SizeS, false, true, -1},
	{"decode/deterministic-no-prefix-m", "zqz4cwzgyw8ks97869", "", SizeM, false, true, -1},
	{"decode/deterministic-upper-marker", "usr_ZCVT7EPAC0T1EBCSJFYF7CWZ25", "usr", SizeXL, false, true, -1},
}

// checkDecoded fails the test when got is not what c expects.
func checkDecoded(t *testing.T, c decodeCase, got Decoded) {
	t.Helper()
	if got.Prefix != c.prefix || got.Size != c.size || got.Compact != c.compact || got.Deterministic != c.deterministic {
		t.Errorf("%s: Decoded = %+v, want prefix %q size %v compact %v deterministic %v",
			c.id, got, c.prefix, c.size, c.compact, c.deterministic)
	}
	switch {
	case c.millis < 0 && !got.Time.IsZero():
		t.Errorf("%s: Time = %v, want the zero time", c.id, got.Time)
	case c.millis >= 0 && (got.Time.UnixMilli() != c.millis || got.Time.Location() != time.UTC):
		t.Errorf("%s: Time = %v (%d ms), want %d ms in UTC", c.id, got.Time, got.Time.UnixMilli(), c.millis)
	}
}

// Every corpus decode case, at both ends of the clock window.
//
// sabotage: in compactTime, subtract the window when t-now >= oneDay; the
// compact cases go red at the window's start.
func TestDecodeCorpusCases(t *testing.T) {
	for _, now := range []int64{windowStart, windowEnd - 1} {
		for _, c := range decodeCases {
			got, err := DecodeWith(c.input, DecodeOptions{Now: time.UnixMilli(now)})
			if err != nil {
				t.Errorf("%s at clock %d: %v", c.id, now, err)
				continue
			}
			checkDecoded(t, c, got)
		}
	}
}

// Just outside the window, each end's case comes back from the wrong 2^40
// ms window: the cases at 1700000000000 ms one millisecond before the start,
// and the compact guard's last valid time at the end.
//
// sabotage: in compactTime, subtract the window when t-now > oneDay+1; the
// start row goes red.
func TestDecodeClockWindowEnds(t *testing.T) {
	rows := []struct {
		name  string
		input string
		now   int64
		want  int64
	}{
		{"one before the start", "usr_hf7yat0004hm5", windowStart - 1, 1_700_000_000_000 - compactWindow},
		{"the end", "usr_yzzzzzzz04hm5", windowEnd, 1_065_151_889_407 + compactWindow},
	}
	for _, r := range rows {
		got, err := DecodeWith(r.input, DecodeOptions{Now: time.UnixMilli(r.now)})
		if err != nil {
			t.Fatalf("%s: %v", r.name, err)
		}
		if got.Time.UnixMilli() != r.want {
			t.Errorf("%s: Time = %d ms, want %d", r.name, got.Time.UnixMilli(), r.want)
		}
	}
}

// Every corpus decode refusal, by its case id, with the reason in a
// *DecodeError whose text is the corpus name.
//
// sabotage: in form, check the alphabet before the length; the
// decode-length-and-character row goes red.
func TestDecodeCorpusRefusals(t *testing.T) {
	cases := []struct {
		id     string
		input  string
		reason Reason
	}{
		{"invalid/decode-empty-string", "", ReasonEmpty},
		{"invalid/decode-ends-in-delimiter", "usr_", ReasonEmpty},
		{"invalid/decode-prefix-with-delimiter-ends-in-delimiter", "a_b_", ReasonEmpty},
		{"invalid/decode-delimiter-only", "_", ReasonEmpty},
		{"invalid/decode-length-11", "usr_01hf7yat000", ReasonInvalidLength},
		{"invalid/decode-length-25", "usr_01hf7yat00000g40r40m30e20", ReasonInvalidLength},
		{"invalid/decode-length-27", "usr_01hf7yat00000g40r40m30e2090", ReasonInvalidLength},
		{"invalid/decode-no-delimiter-short", "usr", ReasonInvalidLength},
		{"invalid/decode-character-in-time", "usr_01hu7yat00000g40r40m30e209", ReasonInvalidCharacter},
		{"invalid/decode-character-in-random", "usr_01hf7yat00000g4lr40m30e209", ReasonInvalidCharacter},
		{"invalid/decode-upper-o-in-random", "usr_01hf7yat00000g40r40mO0e209", ReasonInvalidCharacter},
		{"invalid/decode-hyphen-in-body", "usr_01hf7yat00-00g40r40m30e209", ReasonInvalidCharacter},
		{"invalid/decode-character-in-deterministic", "usr_zcvt7epac0t1ebcsjfyf7cwzi5", ReasonInvalidCharacter},
		{"invalid/decode-non-ascii-table-length", "01hf7yat00000g40r40m30e2é", ReasonInvalidCharacter},
		{"invalid/decode-non-ascii-off-table-length", "01hf7yat00000g40r40m30e20é", ReasonInvalidLength},
		{"invalid/decode-length-and-character", "usr_01hf7yat00u", ReasonInvalidLength},
	}
	for _, c := range cases {
		got, err := DecodeWith(c.input, DecodeOptions{Now: time.UnixMilli(windowStart)})
		var decodeErr *DecodeError
		if !errors.As(err, &decodeErr) {
			t.Errorf("%s: Decode = %+v, %v; want a *DecodeError", c.id, got, err)
			continue
		}
		if decodeErr.Reason != c.reason || err.Error() != string(c.reason) {
			t.Errorf("%s: reason %q, text %q; want %q", c.id, decodeErr.Reason, err.Error(), c.reason)
		}
		if got != (Decoded{}) {
			t.Errorf("%s: Decoded %+v returned with its error", c.id, got)
		}
	}
}

// Every symbol reads as its value in both cases, and no other byte is a
// symbol: the letters I, L, O and U, the bytes beside each range and every
// byte above 0x7f are refused.
//
// sabotage: in symbolValue, end the a-to-h range at i instead; the i and I
// bytes go red.
func TestSymbolValues(t *testing.T) {
	for v := range 32 {
		for _, alphabet := range []string{lowerAlphabet, upperAlphabet} {
			got, ok := symbolValue(alphabet[v])
			if !ok || got != uint64(v) {
				t.Errorf("symbolValue(%q) = %d, %v; want %d", alphabet[v], got, ok, v)
			}
		}
	}
	for b := range 256 {
		in := strings.IndexByte(lowerAlphabet, byte(b)) >= 0 || strings.IndexByte(upperAlphabet, byte(b)) >= 0
		if _, ok := symbolValue(byte(b)); ok != in {
			t.Errorf("symbolValue(0x%02x) ok = %v, want %v", b, ok, in)
		}
	}
}

// The body follows the last delimiter a left-to-right scan finds, matches
// not overlapping, as the reference's split does; an empty Delimiter is
// "_".
//
// sabotage: in splitID, use strings.LastIndex in place of the scan; the
// "---" row goes red.
func TestDecodeDelimiter(t *testing.T) {
	rows := []struct {
		input, delimiter, prefix string
		reason                   Reason
	}{
		{"cus--01hf7yat0004h1", "--", "cus", ""},
		{"a--b--01hf7yat0004h1", "--", "a--b", ""},
		{"usr---01hf7yat00", "--", "", ReasonInvalidLength},
		{"usr---01hf7yat0", "--", "", ReasonInvalidCharacter},
		{"cus_01hf7yat0004h1", "", "cus", ""},
		{"cus_01hf7yat0004h1", "--", "", ReasonInvalidCharacter},
		{"_01hf7yat0004h1", "_", "", ""},
		{"cus.01hf7yat0004h1", ".", "cus", ""},
	}
	for _, r := range rows {
		got, err := DecodeWith(r.input, DecodeOptions{Delimiter: r.delimiter, Now: time.UnixMilli(windowStart)})
		var decodeErr *DecodeError
		switch {
		case r.reason == "" && err != nil:
			t.Errorf("%q under %q: %v", r.input, r.delimiter, err)
		case r.reason != "" && (!errors.As(err, &decodeErr) || decodeErr.Reason != r.reason):
			t.Errorf("%q under %q: %v, want %s", r.input, r.delimiter, err, r.reason)
		case r.reason == "" && got.Prefix != r.prefix:
			t.Errorf("%q under %q: prefix %q, want %q", r.input, r.delimiter, got.Prefix, r.prefix)
		}
	}
}

// A marker body at a compact length decodes as the reference decodes it:
// deterministic, its size and Compact from the length, no time.
//
// sabotage: in DecodeWith, take the marker branch for a standard length
// only; the time is read and the row goes red.
func TestDecodeMarkerAtACompactLength(t *testing.T) {
	got, err := DecodeWith("usr_zzzzzzzzzzzzz", DecodeOptions{Now: time.UnixMilli(windowStart)})
	want := Decoded{Prefix: "usr", Size: SizeS, Compact: true, Deterministic: true}
	if err != nil || got != want {
		t.Fatalf("Decode = %+v, %v; want %+v", got, err, want)
	}
}

// The clock is read only for a compact body: a standard or deterministic
// body decodes the same under any clock, the zero DecodeOptions included.
//
// sabotage: in DecodeWith, take the compact branch also for a clock before
// the epoch; the standard row goes red.
func TestDecodeReadsTheClockOnlyForACompactBody(t *testing.T) {
	for _, now := range []time.Time{{}, time.UnixMilli(0), time.UnixMilli(-1), time.UnixMilli(1 << 50)} {
		got, err := DecodeWith("usr_01hf7yat0004h1", DecodeOptions{Now: now})
		if err != nil || got.Time.UnixMilli() != 1_700_000_000_000 {
			t.Errorf("standard under %v: %+v, %v", now, got, err)
		}
		got, err = DecodeWith("usr_zcvt7epac0t1ebcsjfyf7cwz25", DecodeOptions{Now: now})
		if err != nil || !got.Time.IsZero() {
			t.Errorf("deterministic under %v: %+v, %v", now, got, err)
		}
	}
}

// Decode is DecodeWith with the zero options: the delimiter "_" and the
// system clock. A compact case's time depends on that clock, so only its
// prefix, size and form are checked here; the clock window tests above pin
// its time through an injected clock.
//
// sabotage: in Decode, pass DecodeOptions{Delimiter: "-"}; every row goes
// red.
func TestDecodeDefaults(t *testing.T) {
	for _, c := range decodeCases {
		got, err := Decode(c.input)
		if err != nil {
			t.Errorf("%s: %v", c.id, err)
			continue
		}
		if c.compact {
			if got.Prefix != c.prefix || got.Size != c.size || !got.Compact || got.Deterministic {
				t.Errorf("%s: Decoded = %+v", c.id, got)
			}
			continue
		}
		checkDecoded(t, c, got)
	}
}

// A compact time more than one day ahead of the clock is moved back one
// window whatever the clock, as the wire format's subtraction has no
// exception: under a clock in the first window after the epoch, or before
// the epoch (read as the epoch), that gives a time before the epoch.
//
// sabotage: in compactTime, subtract the window only when t >=
// compactWindow; the two epoch rows and the decode row go red.
func TestCompactTimeNearTheEpoch(t *testing.T) {
	rows := []struct {
		name      string
		low       uint64
		now, want int64
	}{
		{"at the epoch, far ahead", 1 << 39, 0, 1<<39 - compactWindow},
		{"before the epoch, far ahead", 1 << 39, -1_000, 1<<39 - compactWindow},
		{"before the epoch", 5, -1_000, 5},
		{"a day ahead exactly", oneDay, 0, oneDay},
		{"second window, far ahead", 1 << 39, compactWindow, 1 << 39},
		{"second window, behind", 7, compactWindow + 100, compactWindow + 7},
	}
	for _, r := range rows {
		if got := compactTime(r.low, r.now); got != r.want {
			t.Errorf("%s: compactTime(%d, %d) = %d, want %d", r.name, r.low, r.now, got, r.want)
		}
	}
	got, err := DecodeWith("usr_hf7yat0004hm5", DecodeOptions{Now: time.UnixMilli(1_000)})
	if want := int64(600_488_372_224) - compactWindow; err != nil || got.Time.UnixMilli() != want {
		t.Errorf("decode under a clock of 1000 ms: %+v, %v; want %d ms", got, err, want)
	}
}

// IsValid is whether DecodeWith accepts the id under the delimiter, and,
// for a non-empty prefix, whether the id carries exactly that prefix.
//
// sabotage: in IsValid, drop the prefix comparison; the "usr" row goes red.
func TestIsValid(t *testing.T) {
	rows := []struct {
		id, delimiter, prefix string
		want                  bool
	}{
		{"cus_01hf7yat0004h1", "_", "", true},
		{"cus_01hf7yat0004h1", "", "", true},
		{"cus_01hf7yat0004h1", "_", "cus", true},
		{"cus_01hf7yat0004h1", "_", "usr", false},
		{"01hf7yat0004h1", "_", "", true},
		{"01hf7yat0004h1", "_", "cus", false},
		{"cus--01hf7yat0004h1", "--", "cus", true},
		{"cus--01hf7yat0004h1", "_", "", false},
		{"cus_hf7yat0004hm5", "_", "cus", true},
		{"cus_01hf7yat0004h12", "_", "", false},
		{"cus_", "_", "", false},
		{"", "_", "", false},
		{"cus_zt9vnyfd85bff0", "_", "cus", true},
		{"usr_zcvt7epac0t1ebcsjfyf7cwzi5", "_", "", false},
	}
	for _, r := range rows {
		if got := IsValid(r.id, r.delimiter, r.prefix); got != r.want {
			t.Errorf("IsValid(%q, %q, %q) = %v, want %v", r.id, r.delimiter, r.prefix, got, r.want)
		}
	}
}

// IsDeterministic is whether DecodeWith accepts the id under the delimiter
// and its body starts with the marker; a malformed marker body is not one.
//
// sabotage: in IsDeterministic, test only the first body byte; the
// truncated marker row goes red.
func TestIsDeterministic(t *testing.T) {
	rows := []struct {
		id, delimiter string
		want          bool
	}{
		{"usr_zcvt7epac0t1ebcsjfyf7cwz25", "_", true},
		{"usr_ZCVT7EPAC0T1EBCSJFYF7CWZ25", "", true},
		{"zqz4cwzgyw8ks97869", "_", true},
		{"usr--zcvt7epac0t1ebcsjfyf7cwz25", "--", true},
		{"usr--zcvt7epac0t1ebcsjfyf7cwz25", "_", false},
		{"usr_01hf7yat0004h1", "_", false},
		{"usr_zcvt7epac", "_", false},
		{"usr_zcvt7epac0t1ebcsjfyf7cwzi5", "_", false},
		{"", "_", false},
	}
	for _, r := range rows {
		if got := IsDeterministic(r.id, r.delimiter); got != r.want {
			t.Errorf("IsDeterministic(%q, %q) = %v, want %v", r.id, r.delimiter, got, r.want)
		}
	}
}

// An id Generate makes decodes back to its prefix, size, time form and time
// at every size and form whose length the length table reads as itself.
//
// sabotage: in form, read the length 14 as SizeM; every SizeS row goes
// red.
func TestDecodeReadsBackGenerate(t *testing.T) {
	sizes := []Size{SizeXS, SizeS, SizeM, SizeL, SizeXL}
	for _, millis := range []int64{0, 1_700_000_000_000, 1<<48 - 1} {
		for _, size := range sizes {
			id, err := Generate(Options{Prefix: "usr", Size: size, Time: time.UnixMilli(millis), Case: CaseUpper})
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			got, err := DecodeWith(id, DecodeOptions{Now: time.UnixMilli(windowStart)})
			if err != nil || got.Prefix != "usr" || got.Size != size || got.Compact || got.Time.UnixMilli() != millis {
				t.Errorf("%s (%v at %d ms): %+v, %v", id, size, millis, got, err)
			}
		}
	}
	for _, size := range []Size{SizeS, SizeL, SizeXL} {
		id, err := Generate(Options{Size: size, CompactTime: true, Time: corpusTime})
		if err != nil {
			t.Fatalf("Generate: %v", err)
		}
		got, err := DecodeWith(id, DecodeOptions{Now: corpusTime})
		if err != nil || got.Size != size || !got.Compact || !got.Time.Equal(corpusTime) {
			t.Errorf("%s (compact %v): %+v, %v", id, size, got, err)
		}
	}
}

// Decoding never panics, and its answers agree with each other: an error
// is always a *DecodeError with one of the three reasons, a success is a
// body the length table reads, and IsValid and IsDeterministic agree with
// DecodeWith. The seeds are truncated, overlong, wrong-alphabet and empty
// inputs, under the default delimiter and others.
//
// sabotage: in form, return the reason "invalid" for a length off the
// table; the overlong and truncated seeds go red.
func FuzzDecode(f *testing.F) {
	seeds := []struct{ id, delimiter string }{
		{"", ""},
		{"_", "_"},
		{"usr_", "_"},
		{"usr_01hf7yat0004hmasw9nf6yy093", "_"},
		{"usr_01hf7yat0004hmasw9nf6yy09", "_"},
		{"usr_01hf7yat0004hmasw9nf6yy0", "_"},
		{"usr_01hf7yat0", "_"},
		{"usr_01hf7yat0004hmasw9nf6yy0933", "_"},
		{"usr_01hf7yat0004hmasw9nf6yy093" + strings.Repeat("0", 64), "_"},
		{"usr_01hf7yat0004hmasw9nf6yy0il", "_"},
		{"usr_01hf7yat0004hmasw9nf6yy0uo", "_"},
		{"usr_01hf7yat00\x00\xff", "_"},
		{"01hf7yat00000g40r40m30e2é", ""},
		{"usr_hf7yat0004hm5", "_"},
		{"usr_zcvt7epac0t1ebcsjfyf7cwz25", "_"},
		{"usr---01hf7yat0", "--"},
		{"aaa01hf7yat00", "aa"},
		{"\xff\xfe_01hf7yat00", "\xfe"},
	}
	for _, s := range seeds {
		f.Add(s.id, s.delimiter, int64(windowStart))
	}
	f.Add("usr_hf7yat0004hm5", "_", int64(-1<<62))
	f.Add("usr_yzzzzzzz04hm5", "_", int64(1<<62))
	f.Fuzz(func(t *testing.T, id, delimiter string, now int64) {
		got, err := DecodeWith(id, DecodeOptions{Delimiter: delimiter, Now: time.UnixMilli(now)})
		valid := IsValid(id, delimiter, "")
		deterministic := IsDeterministic(id, delimiter)
		if err != nil {
			var decodeErr *DecodeError
			if !errors.As(err, &decodeErr) {
				t.Fatalf("error %T %v is not a *DecodeError", err, err)
			}
			switch decodeErr.Reason {
			case ReasonEmpty, ReasonInvalidLength, ReasonInvalidCharacter:
			default:
				t.Fatalf("reason %q is outside the closed set", decodeErr.Reason)
			}
			if valid || deterministic {
				t.Fatalf("refused (%v) but IsValid %v, IsDeterministic %v", err, valid, deterministic)
			}
			return
		}
		if !valid || deterministic != got.Deterministic {
			t.Fatalf("decoded %+v but IsValid %v, IsDeterministic %v", got, valid, deterministic)
		}
		if got.Deterministic != got.Time.IsZero() {
			t.Fatalf("decoded %+v: Deterministic and a zero Time disagree", got)
		}
	})
}
