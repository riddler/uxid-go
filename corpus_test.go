package uxid

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// The conformance corpus vendored under conformance/ (uxid-ex at the tag
// conformance/SOURCE.json names), run case by case as the corpus README's
// "Running the cases" says: one table test per case file, every line of
// every file the manifest lists, nothing skipped. A case is read strictly
// (see decodeStrict), so a later corpus cannot be half-read, and a failing
// case is reported by its id. corpus_hash_test.go checks the files' hash
// and their case counts.

// corpusDir is the vendored corpus, relative to the package directory the
// tests run in.
const corpusDir = "conformance"

// corpusKinds are the case kinds this runner knows, one file each.
var corpusKinds = []string{"encode", "deterministic", "decode", "invalid", "monotonic"}

// corpusNow is the time passed where a case needs a clock and reads none:
// a generate request with no time and a decode refusal. It is the time most
// cases carry, 1700000000000 ms.
var corpusNow = time.UnixMilli(1_700_000_000_000)

// corpusManifest is manifest.json: the corpus version, the case files in
// order and the corpus hash.
type corpusManifest struct {
	CorpusHash    string       `json:"corpus_hash"`
	CorpusVersion int          `json:"corpus_version"`
	Files         []corpusFile `json:"files"`
}

// corpusFile is one entry of the manifest's files.
type corpusFile struct {
	CaseCount int    `json:"case_count"`
	File      string `json:"file"`
	Kind      string `json:"kind"`
}

// decodeStrict reads data, one JSON value, into v and refuses anything v
// does not hold exactly. The decoder disallows unknown fields; it also
// matches a key to a field without regard to case, so v is then encoded
// again and must read back as the same JSON value as data: a key spelt in
// another case, a key v has no field for, a missing key (encoded again
// with its zero value) and a value v changes all fail.
func decodeStrict(data []byte, v any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		return err
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("more than one JSON value")
	}
	again, err := json.Marshal(v)
	if err != nil {
		return err
	}
	same, err := sameJSON(data, again)
	if err != nil {
		return err
	}
	if !same {
		return fmt.Errorf("does not read back as itself: read %s, holds %s", data, again)
	}
	return nil
}

// sameJSON reports whether a and b are the same JSON value, numbers
// compared by their text.
func sameJSON(a, b []byte) (bool, error) {
	read := func(data []byte) (any, error) {
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.UseNumber()
		var value any
		err := decoder.Decode(&value)
		return value, err
	}
	x, err := read(a)
	if err != nil {
		return false, err
	}
	y, err := read(b)
	if err != nil {
		return false, err
	}
	return reflect.DeepEqual(x, y), nil
}

// readManifest reads conformance/manifest.json strictly.
func readManifest(t *testing.T) corpusManifest {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(corpusDir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest corpusManifest
	if err := decodeStrict(data, &manifest); err != nil {
		t.Fatalf("manifest.json: %v", err)
	}
	return manifest
}

// caseLines splits a case file into its lines, each ended by a newline.
func caseLines(data []byte) ([]string, error) {
	text := string(data)
	if text == "" {
		return nil, nil
	}
	if !strings.HasSuffix(text, "\n") {
		return nil, errors.New("the last line has no newline")
	}
	return strings.Split(strings.TrimSuffix(text, "\n"), "\n"), nil
}

// readCases reads every case of the manifest's one file of kind, each line
// strictly into a C, after checking the manifest's corpus version, the
// file's line count against its case_count and that every case id starts
// with the kind.
func readCases[C interface{ caseID() string }](t *testing.T, kind string) []C {
	t.Helper()
	manifest := readManifest(t)
	if manifest.CorpusVersion != 1 {
		t.Fatalf("corpus_version %d is not one this runner knows", manifest.CorpusVersion)
	}
	var entries []corpusFile
	for _, entry := range manifest.Files {
		if entry.Kind == kind {
			entries = append(entries, entry)
		}
	}
	if len(entries) != 1 {
		t.Fatalf("the manifest lists %d files of kind %q, want 1", len(entries), kind)
	}
	data, err := os.ReadFile(filepath.Join(corpusDir, entries[0].File))
	if err != nil {
		t.Fatal(err)
	}
	lines, err := caseLines(data)
	if err != nil {
		t.Fatalf("%s: %v", entries[0].File, err)
	}
	if len(lines) != entries[0].CaseCount {
		t.Fatalf("%s: %d lines, case_count %d", entries[0].File, len(lines), entries[0].CaseCount)
	}
	cases := make([]C, len(lines))
	for i, line := range lines {
		if err := decodeStrict([]byte(line), &cases[i]); err != nil {
			t.Fatalf("%s line %d: %v", entries[0].File, i+1, err)
		}
		if id := cases[i].caseID(); !strings.HasPrefix(id, kind+"/") {
			t.Fatalf("%s line %d: the id %q does not start with %s/", entries[0].File, i+1, id, kind)
		}
	}
	return cases
}

// corpusSize reads a size in the short spelling the corpus uses.
func corpusSize(text string) (Size, error) {
	switch text {
	case "xs":
		return SizeXS, nil
	case "s":
		return SizeS, nil
	case "m":
		return SizeM, nil
	case "l":
		return SizeL, nil
	case "xl":
		return SizeXL, nil
	}
	return 0, fmt.Errorf("%q is not a size", text)
}

// corpusCase reads a letter case.
func corpusCase(text string) (Case, error) {
	switch text {
	case "lower":
		return CaseLower, nil
	case "upper":
		return CaseUpper, nil
	}
	return 0, fmt.Errorf("%q is not a case", text)
}

// corpusBytes reads lowercase hex; the empty string is an empty, non-nil
// slice, since in the corpus an empty rand is a fixed field of no bytes.
func corpusBytes(text string) ([]byte, error) {
	if text != strings.ToLower(text) {
		return nil, fmt.Errorf("%q is not lowercase hex", text)
	}
	b, err := hex.DecodeString(text)
	if err != nil {
		return nil, fmt.Errorf("%q is not hex: %w", text, err)
	}
	if b == nil {
		b = []byte{}
	}
	return b, nil
}

// prefixOf is a corpus prefix as Options and Decoded spell it: null is no
// prefix, which is the empty string.
func prefixOf(prefix *string) string {
	if prefix == nil {
		return ""
	}
	return *prefix
}

// zeroReader gives zeros: a request a case says is refused reads it, so a
// request that is not refused builds an id and the case fails on it.
type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}

// encodeCase is one line of encode.json.
type encodeCase struct {
	Expected string `json:"expected"`
	ID       string `json:"id"`
	Input    struct {
		Case    string  `json:"case"`
		Compact bool    `json:"compact"`
		Prefix  *string `json:"prefix"`
		Rand    string  `json:"rand"`
		Size    string  `json:"size"`
		Time    int64   `json:"time"`
	} `json:"input"`
}

func (c encodeCase) caseID() string { return c.ID }

// Every encode case: the id built from the input, with the random field
// fixed to rand and the Reader never read, is expected.
//
// sabotage: in writeBytes, write the short last group shifted left
// (alphabet[buffer<<(5-bits)]); every case whose random field ends in a
// partial group goes red.
func TestCorpusEncode(t *testing.T) {
	for _, c := range readCases[encodeCase](t, "encode") {
		t.Run(c.ID, func(t *testing.T) {
			size, err := corpusSize(c.Input.Size)
			if err != nil {
				t.Fatal(err)
			}
			letters, err := corpusCase(c.Input.Case)
			if err != nil {
				t.Fatal(err)
			}
			random, err := corpusBytes(c.Input.Rand)
			if err != nil {
				t.Fatal(err)
			}
			got, err := Generate(Options{
				Prefix:      prefixOf(c.Input.Prefix),
				Size:        size,
				CompactTime: c.Input.Compact,
				Case:        letters,
				Time:        time.UnixMilli(c.Input.Time),
				Rand:        random,
				Reader:      failingReader{t},
			})
			if err != nil || got != c.Expected {
				t.Fatalf("Generate = %q, %v; want %q", got, err, c.Expected)
			}
		})
	}
}

// deterministicCase is one line of deterministic.json.
type deterministicCase struct {
	Expected string `json:"expected"`
	ID       string `json:"id"`
	Input    struct {
		Case   string  `json:"case"`
		From   string  `json:"from"`
		Prefix *string `json:"prefix"`
		Size   string  `json:"size"`
	} `json:"input"`
}

func (c deterministicCase) caseID() string { return c.ID }

// Every deterministic case: the deterministic id of from under the prefix,
// size and case is expected, read with no clock and no Reader.
//
// sabotage: in writeDeterministic, leave the count byte out of the hash
// input; every case goes red.
func TestCorpusDeterministic(t *testing.T) {
	for _, c := range readCases[deterministicCase](t, "deterministic") {
		t.Run(c.ID, func(t *testing.T) {
			size, err := corpusSize(c.Input.Size)
			if err != nil {
				t.Fatal(err)
			}
			letters, err := corpusCase(c.Input.Case)
			if err != nil {
				t.Fatal(err)
			}
			got, err := Generate(Options{
				Prefix: prefixOf(c.Input.Prefix),
				Size:   size,
				Case:   letters,
				From:   new(c.Input.From),
				Reader: failingReader{t},
			})
			if err != nil || got != c.Expected {
				t.Fatalf("Generate = %q, %v; want %q", got, err, c.Expected)
			}
		})
	}
}

// decodeCase is one line of decode.json.
type decodeCase struct {
	Expected struct {
		Compact       bool    `json:"compact"`
		Deterministic bool    `json:"deterministic"`
		Prefix        *string `json:"prefix"`
		Size          string  `json:"size"`
		Time          *int64  `json:"time"`
	} `json:"expected"`
	ID    string `json:"id"`
	Input string `json:"input"`
}

func (c decodeCase) caseID() string { return c.ID }

// want is what decoding the case's input gives: a null prefix is the empty
// one, and a null time, a deterministic body's, is the zero time.Time. Time
// is in UTC.
func (c decodeCase) want() (Decoded, error) {
	size, err := corpusSize(c.Expected.Size)
	if err != nil {
		return Decoded{}, err
	}
	want := Decoded{
		Prefix:        prefixOf(c.Expected.Prefix),
		Size:          size,
		Compact:       c.Expected.Compact,
		Deterministic: c.Expected.Deterministic,
	}
	if c.Expected.Time != nil {
		want.Time = time.UnixMilli(*c.Expected.Time).UTC()
	}
	return want, nil
}

// sameDecoded reports whether got is want, its time the same instant in the
// same location, or both the zero time.
func sameDecoded(got, want Decoded) bool {
	return got.Prefix == want.Prefix && got.Size == want.Size && got.Compact == want.Compact &&
		got.Deterministic == want.Deterministic && got.Time.Equal(want.Time) &&
		got.Time.IsZero() == want.Time.IsZero() && got.Time.Location() == want.Time.Location()
}

// Every decode case reads back its prefix, size, compact flag,
// deterministic flag and time, under a decoder clock at both ends of the
// window the corpus README bounds (windowEnd is exclusive) and inside it.
//
// sabotage: in compactTime, subtract the window when t-now >= oneDay; the
// compact cases at 1700000000000 ms go red at the window's start.
func TestCorpusDecode(t *testing.T) {
	for _, c := range readCases[decodeCase](t, "decode") {
		t.Run(c.ID, func(t *testing.T) {
			want, err := c.want()
			if err != nil {
				t.Fatal(err)
			}
			for _, now := range []int64{windowStart, corpusNow.UnixMilli(), windowEnd - 1} {
				got, err := DecodeWith(c.Input, DecodeOptions{Now: time.UnixMilli(now)})
				if err != nil || !sameDecoded(got, want) {
					t.Errorf("at clock %d: DecodeWith = %+v, %v; want %+v", now, got, err, want)
				}
			}
		})
	}
}

// invalidCase is one line of invalid.json: a string to decode or a
// generate request, never both.
type invalidCase struct {
	ExpectedError string `json:"expected_error"`
	ID            string `json:"id"`
	Input         struct {
		Decode   *string          `json:"decode,omitzero"`
		Generate *generateRequest `json:"generate,omitzero"`
	} `json:"input"`
}

func (c invalidCase) caseID() string { return c.ID }

// generateRequest is an invalid case's generate request, spelt as an
// encode case spells its input, with every option optional: an absent
// option takes its default.
type generateRequest struct {
	Case      optional[string] `json:"case,omitzero"`
	Compact   optional[bool]   `json:"compact,omitzero"`
	From      optional[string] `json:"from,omitzero"`
	Monotonic optional[bool]   `json:"monotonic,omitzero"`
	Prefix    optional[string] `json:"prefix,omitzero"`
	Rand      optional[string] `json:"rand,omitzero"`
	RandSize  optional[int]    `json:"rand_size,omitzero"`
	Size      optional[string] `json:"size,omitzero"`
	Time      optional[int64]  `json:"time,omitzero"`
}

// optional is a key a request may leave out: Set is whether it is there,
// and Value is nil for null.
type optional[T any] struct {
	Set   bool
	Value *T
}

func (o *optional[T]) UnmarshalJSON(data []byte) error {
	o.Set = true
	return json.Unmarshal(data, &o.Value)
}

func (o optional[T]) MarshalJSON() ([]byte, error) { return json.Marshal(o.Value) }

// IsZero reports an absent key, which omitzero leaves out when the request
// is encoded again.
func (o optional[T]) IsZero() bool { return !o.Set }

// value is the option's value, false when it is absent; null is an error,
// since the request's only nullable option is the prefix.
func (o optional[T]) value(name string) (T, bool, error) {
	var zero T
	switch {
	case !o.Set:
		return zero, false, nil
	case o.Value == nil:
		return zero, false, fmt.Errorf("%s is null", name)
	}
	return *o.Value, true, nil
}

// options builds the request's Options, with zeroReader as the Reader,
// and reports whether it asks for monotonic generation.
func (r generateRequest) options() (Options, bool, error) {
	opts := Options{Reader: zeroReader{}}
	if r.Prefix.Value != nil {
		opts.Prefix = *r.Prefix.Value
	}
	if text, ok, err := r.Size.value("size"); err != nil {
		return opts, false, err
	} else if ok {
		if opts.Size, err = corpusSize(text); err != nil {
			return opts, false, err
		}
	}
	if text, ok, err := r.Case.value("case"); err != nil {
		return opts, false, err
	} else if ok {
		if opts.Case, err = corpusCase(text); err != nil {
			return opts, false, err
		}
	}
	compact, _, err := r.Compact.value("compact")
	if err != nil {
		return opts, false, err
	}
	opts.CompactTime = compact
	if millis, ok, err := r.Time.value("time"); err != nil {
		return opts, false, err
	} else if ok {
		opts.Time = time.UnixMilli(millis)
	}
	if width, ok, err := r.RandSize.value("rand_size"); err != nil {
		return opts, false, err
	} else if ok {
		opts.RandSize = new(width)
	}
	if text, ok, err := r.Rand.value("rand"); err != nil {
		return opts, false, err
	} else if ok {
		if opts.Rand, err = corpusBytes(text); err != nil {
			return opts, false, err
		}
	}
	if from, ok, err := r.From.value("from"); err != nil {
		return opts, false, err
	} else if ok {
		opts.From = new(from)
	}
	monotonic, _, err := r.Monotonic.value("monotonic")
	return opts, monotonic, err
}

// generationErrors are the corpus's names for the generation errors.
var generationErrors = map[string]error{
	"compact_time_out_of_range": ErrCompactTimeOutOfRange,
	"rand_width_mismatch":       ErrRandWidthMismatch,
	"rand_size_too_large":       ErrRandSizeTooLarge,
	"from_with_monotonic":       ErrFromWithMonotonic,
	"rand_with_from":            ErrRandWithFrom,
	"rand_with_monotonic":       ErrRandWithMonotonic,
}

// refuse makes the request: Generate, or with monotonic a call to a
// generator NewMonotonic builds from the same options.
func (r generateRequest) refuse() (string, error) {
	opts, monotonic, err := r.options()
	if err != nil {
		return "", fmt.Errorf("reading the request: %w", err)
	}
	if !monotonic {
		return Generate(opts)
	}
	m, err := NewMonotonic(opts)
	if err != nil {
		if m != nil {
			return "", fmt.Errorf("NewMonotonic returned a generator with %w", err)
		}
		return "", err
	}
	return m.NextWith(nil, corpusNow)
}

// Every invalid case is refused with the error expected_error names: a
// string by DecodeWith, with a *DecodeError whose Reason and text are the
// name and no Decoded beside it; a request by Generate, or by a monotonic
// generator, with the sentinel error whose text is the name, returned
// unwrapped and with no id.
//
// sabotage: in form, return ReasonInvalidCharacter for a length off the
// table; every invalid_length case goes red.
func TestCorpusInvalid(t *testing.T) {
	for _, c := range readCases[invalidCase](t, "invalid") {
		t.Run(c.ID, func(t *testing.T) {
			switch {
			case c.Input.Decode != nil && c.Input.Generate == nil:
				got, err := DecodeWith(*c.Input.Decode, DecodeOptions{Now: corpusNow})
				var decodeErr *DecodeError
				if !errors.As(err, &decodeErr) || string(decodeErr.Reason) != c.ExpectedError ||
					err.Error() != c.ExpectedError || got != (Decoded{}) {
					t.Fatalf("DecodeWith = %+v, %v; want the refusal %s", got, err, c.ExpectedError)
				}
			case c.Input.Generate != nil && c.Input.Decode == nil:
				want, ok := generationErrors[c.ExpectedError]
				if !ok {
					t.Fatalf("%q is not a generation error", c.ExpectedError)
				}
				got, err := c.Input.Generate.refuse()
				if err != want || got != "" || err.Error() != c.ExpectedError {
					t.Fatalf("request = %q, %v; want the refusal %s", got, err, c.ExpectedError)
				}
			default:
				t.Fatal("the input holds neither decode nor generate, or both")
			}
		})
	}
}

// monotonicCaseJSON is one line of monotonic.json.
type monotonicCaseJSON struct {
	Expected []string `json:"expected"`
	ID       string   `json:"id"`
	Input    struct {
		Compact      bool     `json:"compact"`
		InitialSeed  uint64   `json:"initial_seed"`
		OverflowSeed *uint64  `json:"overflow_seed"`
		Prefix       *string  `json:"prefix"`
		Size         string   `json:"size"`
		Steps        []uint64 `json:"steps"`
		Times        []int64  `json:"times"`
	} `json:"input"`
}

func (c monotonicCaseJSON) caseID() string { return c.ID }

// corpusDraws replays a monotonic case's draws call by call, as the corpus
// README orders them: the first call's seed is initial_seed; each later
// call's step is the next of steps; the call whose step overflows draws
// overflow_seed as its second draw. Any other draw, and a seed that does
// not fit the field, is an error, which fails the call.
type corpusDraws struct {
	initial  *uint64
	steps    []uint64
	overflow *uint64
	call     int
	drawn    int
}

// begin marks the start of call number call.
func (d *corpusDraws) begin(call int) {
	d.call, d.drawn = call, 0
}

func (d *corpusDraws) Seed(field []byte) error {
	d.drawn++
	var value uint64
	switch {
	case d.call == 0 && d.drawn == 1 && d.initial != nil:
		value, d.initial = *d.initial, nil
	case d.call > 0 && d.drawn == 2 && d.overflow != nil:
		value, d.overflow = *d.overflow, nil
	default:
		return fmt.Errorf("call %d drew a seed as its draw %d, which the case does not give", d.call, d.drawn)
	}
	if len(field) < 8 && value>>(8*len(field)) != 0 {
		return fmt.Errorf("the seed %d does not fit %d bytes", value, len(field))
	}
	putBigEndian(field, value)
	return nil
}

func (d *corpusDraws) Step(int) (uint64, error) {
	d.drawn++
	if d.call == 0 || d.drawn != 1 || len(d.steps) == 0 {
		return 0, fmt.Errorf("call %d drew a step as its draw %d, which the case does not give", d.call, d.drawn)
	}
	step := d.steps[0]
	d.steps = d.steps[1:]
	return step, nil
}

// spent reports whether every draw of the case was read.
func (d *corpusDraws) spent() bool {
	return d.initial == nil && d.overflow == nil && len(d.steps) == 0
}

// Every monotonic case: one generator, lower case, for the prefix, size and
// time form, its draws replayed from the case, called once per time, gives
// the expected ids in order and reads every draw.
//
// sabotage: in NextWith, carry the call's time instead of the last id's
// when the clock moves back (at = millis); the clock-back cases go red.
func TestCorpusMonotonic(t *testing.T) {
	for _, c := range readCases[monotonicCaseJSON](t, "monotonic") {
		t.Run(c.ID, func(t *testing.T) {
			size, err := corpusSize(c.Input.Size)
			if err != nil {
				t.Fatal(err)
			}
			if len(c.Input.Times) != len(c.Expected) {
				t.Fatalf("%d times, %d expected ids", len(c.Input.Times), len(c.Expected))
			}
			m, err := NewMonotonic(Options{Prefix: prefixOf(c.Input.Prefix), Size: size, CompactTime: c.Input.Compact})
			if err != nil {
				t.Fatalf("NewMonotonic: %v", err)
			}
			draws := &corpusDraws{initial: new(c.Input.InitialSeed), steps: c.Input.Steps, overflow: c.Input.OverflowSeed}
			for call, at := range c.Input.Times {
				draws.begin(call)
				got, err := m.NextWith(draws, time.UnixMilli(at))
				if err != nil || got != c.Expected[call] {
					t.Fatalf("call %d at %d: %q, %v; want %q", call, at, got, err, c.Expected[call])
				}
			}
			if !draws.spent() {
				t.Fatalf("draws left unread: initial %v, steps %v, overflow %v", draws.initial, draws.steps, draws.overflow)
			}
		})
	}
}

// The manifest lists exactly the kinds this runner knows, each once, and
// every case id in the corpus is unique, so every file is run by one of
// the tests above and none is passed over.
//
// sabotage: in corpusKinds, drop "monotonic"; this test goes red.
func TestCorpusManifestListsTheKindsTheRunnerKnows(t *testing.T) {
	manifest := readManifest(t)
	var kinds []string
	for _, entry := range manifest.Files {
		kinds = append(kinds, entry.Kind)
	}
	if !reflect.DeepEqual(kinds, corpusKinds) {
		t.Fatalf("the manifest lists the kinds %v, the runner knows %v", kinds, corpusKinds)
	}
	seen := map[string]bool{}
	for _, entry := range manifest.Files {
		data, err := os.ReadFile(filepath.Join(corpusDir, entry.File))
		if err != nil {
			t.Fatal(err)
		}
		lines, err := caseLines(data)
		if err != nil {
			t.Fatalf("%s: %v", entry.File, err)
		}
		for _, line := range lines {
			var c struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal([]byte(line), &c); err != nil {
				t.Fatalf("%s: %v", entry.File, err)
			}
			if seen[c.ID] {
				t.Errorf("%s: the case id %q is not unique", entry.File, c.ID)
			}
			seen[c.ID] = true
		}
	}
}

// A case line the runner does not hold exactly is refused: an unknown key
// at the top or inside the input, a known key spelt in another case, a
// missing key and a second JSON value.
//
// sabotage: in decodeStrict, drop the read-back comparison; the
// other-case and missing-key rows go red.
func TestCorpusCasesAreReadStrictly(t *testing.T) {
	const good = `{"expected":"usr_01hf7yat00","id":"encode/x","input":{"case":"lower","compact":false,"prefix":"usr","rand":"","size":"xs","time":1700000000000}}`
	var c encodeCase
	if err := decodeStrict([]byte(good), &c); err != nil {
		t.Fatalf("the good line: %v", err)
	}
	rows := map[string]string{
		"unknown key in input": `{"expected":"usr_01hf7yat00","id":"encode/x","input":{"case":"lower","compact":false,"delimiter":"-","prefix":"usr","rand":"","size":"xs","time":1700000000000}}`,
		"unknown key at top":   `{"expected":"usr_01hf7yat00","id":"encode/x","input":{"case":"lower","compact":false,"prefix":"usr","rand":"","size":"xs","time":1700000000000},"note":"x"}`,
		"key in another case":  `{"expected":"usr_01hf7yat00","id":"encode/x","input":{"Case":"lower","compact":false,"prefix":"usr","rand":"","size":"xs","time":1700000000000}}`,
		"missing key":          `{"expected":"usr_01hf7yat00","id":"encode/x","input":{"case":"lower","compact":false,"prefix":"usr","rand":"","size":"xs"}}`,
		"a second value":       good + ` {}`,
	}
	for name, line := range rows {
		var c encodeCase
		if err := decodeStrict([]byte(line), &c); err == nil {
			t.Errorf("%s: read with no error", name)
		}
	}

	// An absent option and a null prefix in a generate request read back as
	// themselves, so the request's optional keys are read strictly too.
	request := `{"expected_error":"rand_width_mismatch","id":"invalid/x","input":{"generate":{"prefix":null,"rand":"01","size":"s"}}}`
	var invalid invalidCase
	if err := decodeStrict([]byte(request), &invalid); err != nil {
		t.Fatalf("a request with a null prefix: %v", err)
	}
	if got, err := invalid.Input.Generate.refuse(); err != ErrRandWidthMismatch {
		t.Fatalf("a request with a null prefix: %q, %v", got, err)
	}
}

// A monotonic case is replayed call by call: a generator that draws a seed
// where the case gives a step, or leaves a draw unread, fails the case,
// and a case whose ids differ fails on the first id that differs.
//
// sabotage: in corpusDraws.Seed, hand out the overflow seed at any draw;
// the seed-in-place-of-a-step row goes red.
func TestCorpusDrawsAreReplayedInOrder(t *testing.T) {
	field := make([]byte, 2)
	draws := &corpusDraws{initial: new(uint64(1)), steps: []uint64{1}, overflow: new(uint64(2))}
	draws.begin(0)
	if err := draws.Seed(field); err != nil {
		t.Fatalf("the first seed: %v", err)
	}
	draws.begin(1)
	if err := draws.Seed(field); err == nil {
		t.Error("a seed in place of a step: no error")
	}
	draws.begin(1)
	if _, err := draws.Step(2); err != nil {
		t.Fatalf("the step: %v", err)
	}
	if draws.spent() {
		t.Error("spent with the overflow seed unread")
	}
	draws.begin(0)
	if _, err := draws.Step(2); err == nil {
		t.Error("a step on the first call: no error")
	}
	wide := &corpusDraws{initial: new(uint64(1 << 16))}
	wide.begin(0)
	if err := wide.Seed(field); err == nil {
		t.Error("a seed wider than the field: no error")
	}
}
