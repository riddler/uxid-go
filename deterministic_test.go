package uxid

import (
	"crypto/sha256"
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"
)

// expectedDeterministic is an oracle for the spec's layout written apart
// from the package's own code: SHA-256 over prefix, 0x00, from, 0x00 and the
// count byte, read as one big integer, its top count x 5 bits written as
// count characters after the marker.
func expectedDeterministic(prefix, from string, count int) string {
	input := prefix + "\x00" + from + "\x00" + string(rune(count))
	digest := sha256.Sum256([]byte(input))
	value := new(big.Int).SetBytes(digest[:])
	value.Rsh(value, uint(256-5*count))
	body := make([]byte, count)
	for i := count - 1; i >= 0; i-- {
		body[i] = lowerAlphabet[new(big.Int).And(value, big.NewInt(31)).Int64()]
		value.Rsh(value, 5)
	}
	return "z" + string(body)
}

// The hash input is the spec's layout at every size, also for an empty
// input string, which is an input like any other and not "no From".
//
// sabotage: in writeDeterministic, drop the zero byte between the prefix
// and from; every row goes red.
func TestGenerateDeterministicLayout(t *testing.T) {
	counts := map[Size]int{SizeXS: 9, SizeS: 13, SizeM: 17, SizeL: 21, SizeXL: 25}
	for size, count := range counts {
		for _, input := range []struct{ prefix, from string }{
			{"usr", "alice@example.com"},
			{"", ""},
			{"cus", ""},
			{"", "acme-corp"},
		} {
			got, err := Generate(Options{Prefix: input.prefix, Size: size, From: new(input.from)})
			if err != nil {
				t.Fatalf("Generate(%v, %q, %q): %v", size, input.prefix, input.from, err)
			}
			want := expectedDeterministic(input.prefix, input.from, count)
			if input.prefix != "" {
				want = input.prefix + "_" + want
			}
			if got != want {
				t.Errorf("Generate(%v, %q, %q) = %q, want %q", size, input.prefix, input.from, got, want)
			}
		}
	}
}

// A deterministic id reads none of the time-based fields: no clock, no
// Reader, no random width and no time form, so options that a time-based
// id would refuse give the same id.
//
// sabotage: in Generate, move the From branch below the time check; the
// out-of-range time row goes red.
func TestGenerateDeterministicIgnoresTheTimeFields(t *testing.T) {
	const want = "usr_zcvt7epac0t1ebcsjfyf7cwz25"
	cases := map[string]Options{
		"a time out of range":    {Time: time.UnixMilli(-1)},
		"compact time":           {CompactTime: true, Time: time.UnixMilli(2_164_663_517_184)},
		"a random width over 10": {RandSize: new(11)},
		"a negative width":       {RandSize: new(-1)},
	}
	for name, opts := range cases {
		opts.Prefix = "usr"
		opts.From = new("alice@example.com")
		opts.Reader = failingReader{t}
		got, err := Generate(opts)
		if err != nil || got != want {
			t.Errorf("%s: Generate = %q, %v; want %q", name, got, err, want)
		}
	}
}

// The delimiter goes between the prefix and a deterministic body as it
// does for a time-based one, and is not part of the hash input.
//
// sabotage: in Generate's From branch, write the prefix with
// defaultDelimiter in place of delimiter; the "-" row goes red.
func TestGenerateDeterministicDelimiter(t *testing.T) {
	for delimiter, want := range map[string]string{
		"-":  "usr-zcvt7epac0t1ebcsjfyf7cwz25",
		"::": "usr::zcvt7epac0t1ebcsjfyf7cwz25",
	} {
		got, err := Generate(Options{Prefix: "usr", Delimiter: delimiter, From: new("alice@example.com")})
		if err != nil || got != want {
			t.Errorf("Generate(%q) = %q, %v; want %q", delimiter, got, err, want)
		}
	}
}

// Fixed bytes with From are refused (the corpus runner runs the corpus's
// own case), also an empty fixed field, which is still fixed bytes, and
// fixed bytes with From under an unknown Size or Case, which the
// reference's order refuses for the fixed bytes first.
//
// sabotage: in Generate, move the Rand-with-From check below the Size and
// Case checks; the unknown Size and unknown Case rows go red.
func TestGenerateRandWithFrom(t *testing.T) {
	cases := map[string]Options{
		"a fixed field":        {Prefix: "cus", Size: SizeS, From: new("a"), Rand: []byte{1, 2}},
		"an empty fixed field": {Prefix: "usr", From: new("alice@example.com"), Rand: []byte{}},
		"an unknown Size":      {Size: Size(5), From: new("a"), Rand: []byte{1}},
		"an unknown Case":      {Case: Case(2), From: new("a"), Rand: []byte{1}},
	}
	for name, opts := range cases {
		got, err := Generate(opts)
		if !errors.Is(err, ErrRandWithFrom) || got != "" {
			t.Errorf("%s: Generate = %q, %v; want ErrRandWithFrom", name, got, err)
		}
		if err != nil && err.Error() != "rand_with_from" {
			t.Errorf("%s: error text %q, want the corpus name", name, err.Error())
		}
	}
}

// A deterministic request checks its Size and Case as a time-based one
// does.
//
// sabotage: in Generate, move the Size check below the From branch; the
// Size(5) row goes red.
func TestGenerateDeterministicUnknownSizeAndCase(t *testing.T) {
	if _, err := Generate(Options{Size: Size(5), From: new("a")}); !errors.Is(err, ErrUnknownSize) {
		t.Errorf("Size(5): %v, want ErrUnknownSize", err)
	}
	if _, err := Generate(Options{Case: Case(2), From: new("a")}); !errors.Is(err, ErrUnknownCase) {
		t.Errorf("Case(2): %v, want ErrUnknownCase", err)
	}
}

// Every deterministic body sorts after every time-based body under one
// prefix and one case: the marker is the highest symbol.
//
// sabotage: in writeDeterministic, write alphabet[0] as the marker; both
// rows go red.
func TestDeterministicSortsAfterTimeBased(t *testing.T) {
	deterministic, err := Generate(Options{Prefix: "usr", Size: SizeXS, From: new("")})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	for _, opts := range []Options{
		{Size: SizeXS, Time: time.UnixMilli(1<<48 - 1)},
		{Size: SizeS, CompactTime: true, Time: time.UnixMilli(1_065_151_889_407), Rand: []byte{0xff, 0xff, 0xff}},
	} {
		opts.Prefix = "usr"
		timeBased, err := Generate(opts)
		if err != nil {
			t.Fatalf("Generate: %v", err)
		}
		if strings.Compare(timeBased, deterministic) >= 0 {
			t.Errorf("time-based %q does not sort before deterministic %q", timeBased, deterministic)
		}
	}
}
