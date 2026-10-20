package uxid

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// The decoder clock window the vendored corpus's README states
// (conformance/README.md): every compact decode case holds for a clock from
// 1699913600000 ms (2023-11-13T22:13:20.000Z) inclusive to 2164577117183 ms
// (2038-08-04T23:25:17.183Z) exclusive. The start is one day before the
// cases at 1700000000000 ms; the end is 2^40 + 1065151889407 less one day,
// the last clock at which the compact guard's last valid time from the
// window before is still more than one day ahead.
const (
	windowStart int64 = 1_699_913_600_000
	windowEnd   int64 = 2_164_577_117_183
)

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
// system clock, over every decode case of the vendored corpus. A compact
// case's time depends on that clock, so only its prefix, size and form are
// checked here; the corpus runner and the clock window tests above pin its
// time through an injected clock.
//
// sabotage: in Decode, pass DecodeOptions{Delimiter: "-"}; every case goes
// red.
func TestDecodeDefaults(t *testing.T) {
	for _, c := range readCases[decodeCase](t, "decode") {
		want, err := c.want()
		if err != nil {
			t.Fatalf("%s: %v", c.ID, err)
		}
		got, err := Decode(c.Input)
		if err != nil {
			t.Errorf("%s: %v", c.ID, err)
			continue
		}
		if want.Compact {
			got.Time, want.Time = time.Time{}, time.Time{}
		}
		if !sameDecoded(got, want) {
			t.Errorf("%s: Decoded = %+v, want %+v", c.ID, got, want)
		}
	}
}

// A compact time more than one day ahead of the clock is moved back one
// window whatever the clock, as the wire format's subtraction has no
// exception: under a clock in the first window after the epoch that gives
// a time before the epoch. A clock before the epoch is in the window that
// ends at the epoch, by the reference's arithmetic shift, so every compact
// time under it is read from that window.
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
		{"before the epoch", 5, -1_000, 5 - compactWindow},
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

// A clock before the Unix epoch is read with the reference's arithmetic,
// not as the epoch: its bits above bit 40, shifted arithmetically, are -1,
// so a compact time of 5 ms under a clock of -1000 ms is 5 - 2^40 ms. Read
// as the epoch, the clock would give 5 ms.
//
// sabotage: in compactTime, read a clock before the epoch as the epoch
// (now = 0 when now < 0); both rows go red.
func TestDecodeWithAClockBeforeTheEpoch(t *testing.T) {
	rows := []struct {
		name string
		now  int64
		want int64
	}{
		{"one second before", -1_000, 5 - compactWindow},
		{"one window before", -compactWindow, 5 - compactWindow},
		{"just past one window before", -compactWindow - 1, 5 - 2*compactWindow},
	}
	for _, r := range rows {
		got, err := DecodeWith("usr_0000000500000", DecodeOptions{Now: time.UnixMilli(r.now)})
		if err != nil || !got.Compact || got.Time.UnixMilli() != r.want {
			t.Errorf("%s: %+v, %v; want %d ms", r.name, got, err, r.want)
		}
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
