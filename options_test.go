package uxid

import (
	"errors"
	"testing"
)

// sabotage: in ParseSize, map "medium" to SizeL; this test goes red.
func TestParseSize(t *testing.T) {
	spellings := map[string]Size{
		"xs": SizeXS, "xsmall": SizeXS,
		"s": SizeS, "small": SizeS,
		"m": SizeM, "medium": SizeM,
		"l": SizeL, "large": SizeL,
		"xl": SizeXL, "xlarge": SizeXL,
	}
	for text, want := range spellings {
		got, err := ParseSize(text)
		if err != nil || got != want {
			t.Errorf("ParseSize(%q) = %v, %v; want %v", text, got, err, want)
		}
	}
	for _, text := range []string{"", "XL", "Small", "huge", "x", " xl"} {
		if _, err := ParseSize(text); !errors.Is(err, ErrUnknownSize) {
			t.Errorf("ParseSize(%q): %v, want ErrUnknownSize", text, err)
		}
	}
}

// sabotage: in Size.String, return "xlarge" for SizeXL; this test goes red.
func TestSizeStringRoundTrips(t *testing.T) {
	for _, size := range []Size{SizeXS, SizeS, SizeM, SizeL, SizeXL} {
		back, err := ParseSize(size.String())
		if err != nil || back != size {
			t.Errorf("ParseSize(%q) = %v, %v; want %v", size.String(), back, err, size)
		}
		if len(size.String()) > 2 {
			t.Errorf("%v.String() = %q, want the short spelling", int(size), size.String())
		}
	}
	if got := Size(9).String(); got != "Size(9)" {
		t.Errorf("Size(9).String() = %q", got)
	}
}

// sabotage: start the Size constants at iota + 1, so that no constant is
// the zero value; this test goes red.
func TestZeroValues(t *testing.T) {
	var opts Options
	if opts.Size != SizeXL {
		t.Errorf("the zero Size is %v, want xl", opts.Size)
	}
	if opts.Case != CaseLower {
		t.Errorf("the zero Case is %d, want CaseLower", int(opts.Case))
	}
}
