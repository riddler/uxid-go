package uxid

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// replayDraws replays known draws: each Seed takes the next of seeds,
// written big-endian into the field, and each Step the next of steps. It
// records every draw it gives, in order, in calls.
type replayDraws struct {
	seeds []uint64
	steps []uint64
	calls []draw
}

// draw is one draw a generator made: a seed of width bytes, or a step.
type draw struct {
	seed  bool
	width int
	value uint64
}

func (d *replayDraws) Seed(field []byte) error {
	if len(d.seeds) == 0 {
		return errors.New("no seed left to replay")
	}
	value := d.seeds[0]
	d.seeds = d.seeds[1:]
	putBigEndian(field, value)
	d.calls = append(d.calls, draw{seed: true, width: len(field), value: value})
	return nil
}

func (d *replayDraws) Step(width int) (uint64, error) {
	if len(d.steps) == 0 {
		return 0, errors.New("no step left to replay")
	}
	value := d.steps[0]
	d.steps = d.steps[1:]
	d.calls = append(d.calls, draw{width: width, value: value})
	return value, nil
}

// putBigEndian writes value into field as a big-endian unsigned integer.
func putBigEndian(field []byte, value uint64) {
	for i := len(field) - 1; i >= 0; i-- {
		field[i] = byte(value)
		value >>= 8
	}
}

// readerBytes is what a byte source must give for the default draws to
// make calls: a seed as its width in bytes, big-endian, and a step as
// step - 1 in ceil(width / 2) bytes, big-endian.
func readerBytes(calls []draw) []byte {
	var out []byte
	for _, c := range calls {
		if c.seed {
			field := make([]byte, c.width)
			putBigEndian(field, c.value)
			out = append(out, field...)
			continue
		}
		field := make([]byte, (c.width+1)/2)
		putBigEndian(field, c.value-1)
		out = append(out, field...)
	}
	return out
}

// monotonicCase is one case of the vendored corpus's monotonic.json: one
// generator's calls and draws, and the ids it emits. A null prefix is the
// empty one.
type monotonicCase struct {
	id       string
	prefix   string
	size     Size
	compact  bool
	times    []int64
	initial  uint64
	steps    []uint64
	overflow *uint64
	expected []string
}

// corpusMonotonicCases are the cases of the vendored corpus's
// monotonic.json, read through the corpus runner's readCases.
func corpusMonotonicCases(t *testing.T) []monotonicCase {
	t.Helper()
	var cases []monotonicCase
	for _, c := range readCases[monotonicCaseJSON](t, "monotonic") {
		size, err := corpusSize(c.Input.Size)
		if err != nil {
			t.Fatalf("%s: %v", c.ID, err)
		}
		cases = append(cases, monotonicCase{
			id:       c.ID,
			prefix:   prefixOf(c.Input.Prefix),
			size:     size,
			compact:  c.Input.Compact,
			times:    c.Input.Times,
			initial:  c.Input.InitialSeed,
			steps:    c.Input.Steps,
			overflow: c.Input.OverflowSeed,
			expected: c.Expected,
		})
	}
	return cases
}

// corpusMonotonicCase is the vendored monotonic case with the id, failing
// the test when the corpus has none.
func corpusMonotonicCase(t *testing.T, id string) monotonicCase {
	t.Helper()
	for _, c := range corpusMonotonicCases(t) {
		if c.id == id {
			return c
		}
	}
	t.Fatalf("the vendored corpus has no case %s", id)
	return monotonicCase{}
}

// replay is the case's draws: the initial seed, then the overflow seed
// when the case has one, and its steps.
func (c monotonicCase) replay() *replayDraws {
	seeds := []uint64{c.initial}
	if c.overflow != nil {
		seeds = append(seeds, *c.overflow)
	}
	return &replayDraws{seeds: seeds, steps: slices.Clone(c.steps)}
}

// run makes one id per time of the case from m, with draws, and fails the
// test on the first id that is not the expected one.
func (c monotonicCase) run(t *testing.T, m *Monotonic, draws Draws) {
	t.Helper()
	for i, at := range c.times {
		got, err := m.NextWith(draws, time.UnixMilli(at))
		if err != nil || got != c.expected[i] {
			t.Fatalf("call %d at %d: %q, %v; want %q", i, at, got, err, c.expected[i])
		}
	}
}

// The vendored corpus's monotonic cases, which the corpus runner replays
// through a Draws, made again through the default draws, with the byte
// source giving the bytes those draws read: a seed is the field's bytes,
// and a step is step - 1 in the low 4w bits of ceil(w / 2) bytes. Every
// byte is read.
//
// sabotage: in readerDraws.Step, return the masked value without the + 1;
// every case with a step goes red.
func TestMonotonicCorpusCasesThroughTheByteSource(t *testing.T) {
	for _, c := range corpusMonotonicCases(t) {
		t.Run(c.id, func(t *testing.T) {
			recorded := c.replay()
			m, err := NewMonotonic(Options{Prefix: c.prefix, Size: c.size, CompactTime: c.compact})
			if err != nil {
				t.Fatalf("NewMonotonic: %v", err)
			}
			c.run(t, m, recorded)

			source := bytes.NewReader(readerBytes(recorded.calls))
			m, err = NewMonotonic(Options{Prefix: c.prefix, Size: c.size, CompactTime: c.compact, Reader: source})
			if err != nil {
				t.Fatalf("NewMonotonic: %v", err)
			}
			c.run(t, m, nil)
			if source.Len() != 0 {
				t.Fatalf("%d bytes left unread", source.Len())
			}
		})
	}
}

// The default step keeps the low 4w bits of ceil(w / 2) bytes and adds
// one, for every width: all zeros is the step 1, all ones the step
// 2^(4w), and the bits above the mask never count.
//
// sabotage: in readerDraws.Step, mask with 1<<(4*width+1) - 1; every odd
// width goes red.
func TestMonotonicDefaultStepIsMasked(t *testing.T) {
	for width := 1; width <= maxRandBytes; width++ {
		count := (width + 1) / 2
		ones := readerDraws{reader: bytes.NewReader(bytes.Repeat([]byte{0xff}, count))}
		if got, err := ones.Step(width); err != nil || got != 1<<(4*width) {
			t.Errorf("width %d, all ones: %d, %v; want %d", width, got, err, uint64(1)<<(4*width))
		}
		zeros := readerDraws{reader: bytes.NewReader(make([]byte, count))}
		if got, err := zeros.Step(width); err != nil || got != 1 {
			t.Errorf("width %d, all zeros: %d, %v; want 1", width, got, err)
		}
	}
}

// The monotonic refusals, the wire format's and the package's own, all at
// NewMonotonic, which returns no generator with them. With both From and
// Rand the request is refused for From. The corpus runner runs the
// corpus's own monotonic refusals.
//
// sabotage: in NewMonotonic, check Rand before From; the both row goes
// red.
func TestNewMonotonicRefusals(t *testing.T) {
	rows := []struct {
		id       string
		opts     Options
		expected error
	}{
		{"from", Options{Prefix: "cus", From: new("")}, ErrFromWithMonotonic},
		{"rand", Options{Prefix: "cus", Size: SizeXL, Rand: make([]byte, 10)}, ErrRandWithMonotonic},
		{"from and rand", Options{From: new("a"), Rand: []byte{}}, ErrFromWithMonotonic},
		{"unknown size", Options{Size: Size(9)}, ErrUnknownSize},
		{"unknown case", Options{Case: Case(2)}, ErrUnknownCase},
		{"negative rand size", Options{RandSize: new(-1)}, ErrNegativeRandSize},
		{"rand size 11", Options{RandSize: new(11)}, ErrRandSizeTooLarge},
		{"rand size 0", Options{RandSize: new(0)}, ErrEmptyRandWithMonotonic},
		{"xs with rand size 0", Options{Size: SizeXS, RandSize: new(0)}, ErrEmptyRandWithMonotonic},
		{"time before the epoch", Options{Time: time.UnixMilli(-1)}, ErrTimeOutOfRange},
		{"time past the field", Options{Time: time.UnixMilli(maxTime + 1)}, ErrTimeOutOfRange},
	}
	for _, r := range rows {
		t.Run(r.id, func(t *testing.T) {
			m, err := NewMonotonic(r.opts)
			if err != r.expected || m != nil {
				t.Fatalf("NewMonotonic = %v, %v; want nil, %v", m, err, r.expected)
			}
		})
	}
}

// A monotonic xs uses compact time whatever CompactTime says, so it has a
// one-byte field to step: the vendored corpus's compact xs case comes out
// the same with CompactTime false. An explicit RandSize still sets the
// width.
//
// sabotage: in NewMonotonic, take compact time from opts.CompactTime
// alone; NewMonotonic refuses the xs and the test goes red.
func TestMonotonicXSIsCompact(t *testing.T) {
	c := corpusMonotonicCase(t, "monotonic/xs-compact-overflow-clock-back")
	m, err := NewMonotonic(Options{Prefix: c.prefix, Size: SizeXS})
	if err != nil {
		t.Fatalf("NewMonotonic: %v", err)
	}
	c.run(t, m, c.replay())

	m, err = NewMonotonic(Options{Prefix: "usr", Size: SizeXS, RandSize: new(2), Time: corpusTime})
	if err != nil {
		t.Fatalf("NewMonotonic with RandSize: %v", err)
	}
	got, err := m.NextWith(&replayDraws{seeds: []uint64{0}}, time.Time{})
	if want := "usr_hf7yat000000"; err != nil || got != want {
		t.Fatalf("xs with RandSize 2: %q, %v; want %q", got, err, want)
	}
}

// One generator shared by many goroutines gives every one of them strictly
// increasing ids, and no two goroutines the same id, under the race
// detector. A fixed time keeps every call in one millisecond, so the calls
// step the field and overflow it into later milliseconds.
//
// sabotage: in NextWith, drop the m.mu.Lock and its deferred Unlock; the
// race detector fails the test.
func TestMonotonicIsSafeForConcurrentUse(t *testing.T) {
	m, err := NewMonotonic(Options{Prefix: "usr", Size: SizeS, Time: corpusTime})
	if err != nil {
		t.Fatalf("NewMonotonic: %v", err)
	}
	const goroutines, perGoroutine = 16, 400
	ids := make([][]string, goroutines)
	var wg sync.WaitGroup
	for g := range goroutines {
		wg.Go(func() {
			for range perGoroutine {
				id, err := m.Next()
				if err != nil {
					t.Errorf("goroutine %d: %v", g, err)
					return
				}
				ids[g] = append(ids[g], id)
			}
		})
	}
	wg.Wait()

	seen := make(map[string]bool, goroutines*perGoroutine)
	for g, list := range ids {
		for i, id := range list {
			if i > 0 && id <= list[i-1] {
				t.Fatalf("goroutine %d: %q after %q", g, id, list[i-1])
			}
			if seen[id] {
				t.Fatalf("goroutine %d: %q given twice", g, id)
			}
			seen[id] = true
		}
	}
	if len(seen) != goroutines*perGoroutine {
		t.Fatalf("%d ids, want %d", len(seen), goroutines*perGoroutine)
	}
}

// A step that overflows the field in the last millisecond a time field
// holds has no millisecond after it to move to.
//
// sabotage: in NextWith, drop the maxTime check before at++; the id comes
// back with a time past the field and the test goes red.
func TestMonotonicOverflowAtTheLastMillisecond(t *testing.T) {
	m, err := NewMonotonic(Options{Prefix: "usr", Size: SizeS, RandSize: new(1)})
	if err != nil {
		t.Fatalf("NewMonotonic: %v", err)
	}
	last := time.UnixMilli(maxTime)
	draws := &replayDraws{seeds: []uint64{255, 0}, steps: []uint64{1}}
	if got, err := m.NextWith(draws, last); err != nil || got != "usr_7zzzzzzzzzz7" {
		t.Fatalf("first: %q, %v", got, err)
	}
	if got, err := m.NextWith(draws, last); err != ErrTimeOutOfRange {
		t.Fatalf("overflow: %q, %v; want %v", got, err, ErrTimeOutOfRange)
	}
}

// A call's own time outside the time field is refused, under the default
// clock too.
//
// sabotage: in NextWith, drop the inTimeRange check; both rows go red.
func TestMonotonicCallTimeRange(t *testing.T) {
	m, err := NewMonotonic(Options{Prefix: "usr"})
	if err != nil {
		t.Fatalf("NewMonotonic: %v", err)
	}
	for _, at := range []time.Time{time.UnixMilli(-1), time.UnixMilli(maxTime + 1)} {
		if got, err := m.NextWith(&replayDraws{seeds: []uint64{0}}, at); err != ErrTimeOutOfRange {
			t.Errorf("at %v: %q, %v; want %v", at, got, err, ErrTimeOutOfRange)
		}
	}
}

// A compact time whose first character would be the marker is refused,
// both as the call's time and when an overflow moves the id onto it. From
// 255 in a one-byte field every step overflows, so here the refusal
// repeats, and the generator keeps the last id it gave.
//
// sabotage: in NextWith, run the compact guard on the call's time rather
// than the id's (millis in place of at); the overflow row goes red.
func TestMonotonicCompactGuard(t *testing.T) {
	guard := time.UnixMilli(31 << 35)
	m, err := NewMonotonic(Options{Prefix: "usr", Size: SizeXS})
	if err != nil {
		t.Fatalf("NewMonotonic: %v", err)
	}
	if got, err := m.NextWith(&replayDraws{seeds: []uint64{0}}, guard); err != ErrCompactTimeOutOfRange {
		t.Fatalf("at the guard: %q, %v", got, err)
	}

	before := guard.Add(-time.Millisecond)
	draws := &replayDraws{seeds: []uint64{255, 0, 0}, steps: []uint64{1, 1, 1}}
	first, err := m.NextWith(draws, before)
	if err != nil {
		t.Fatalf("before the guard: %v", err)
	}
	for i := range 2 {
		if got, err := m.NextWith(draws, before); err != ErrCompactTimeOutOfRange {
			t.Fatalf("overflow %d onto the guard: %q, %v", i, got, err)
		}
	}
	if m.lastTime != uint64(before.UnixMilli()) || m.last[0] != 255 {
		t.Fatalf("state moved after a refusal: %d, %d; first id %q", m.lastTime, m.last[0], first)
	}
}

// A failed draw returns an error that wraps the draw's own error, and is
// none of the sentinel errors; the generator is left as it was: a failed
// first call leaves no last id, so an earlier time still seeds afresh,
// and a failed step leaves the last id to step from.
//
// sabotage: in NextWith, record the call's time as the last id's before
// the seed is drawn (m.started, m.lastTime = true, at); the seed row goes
// red.
func TestMonotonicFailedDrawLeavesTheState(t *testing.T) {
	m, err := NewMonotonic(Options{Prefix: "usr", Size: SizeS, Time: corpusTime})
	if err != nil {
		t.Fatalf("NewMonotonic: %v", err)
	}
	later := corpusTime.Add(5 * time.Millisecond)
	if got, err := m.NextWith(&replayDraws{}, later); err == nil || !strings.Contains(err.Error(), "no seed left") {
		t.Fatalf("seed failure: %q, %v", got, err)
	}
	if got, err := m.NextWith(&replayDraws{seeds: []uint64{7}}, time.Time{}); err != nil || got != "usr_01hf7yat000031" {
		t.Fatalf("seed after a failure: %q, %v", got, err)
	}
	if got, err := m.NextWith(&replayDraws{}, time.Time{}); err == nil || !strings.Contains(err.Error(), "no step left") {
		t.Fatalf("step failure: %q, %v", got, err)
	}
	got, err := m.NextWith(&replayDraws{steps: []uint64{1}}, time.Time{})
	if err != nil || got != "usr_01hf7yat000040" {
		t.Fatalf("step after a failure: %q, %v", got, err)
	}
}

// The default draws wrap the byte source's error, as Generate does: io.EOF
// for a source with no byte, for a seed and for a step.
//
// sabotage: in readerDraws.Seed, format the error with %v in place of %w;
// the seed row goes red.
func TestMonotonicWrapsAByteSourceFailure(t *testing.T) {
	m, err := NewMonotonic(Options{Prefix: "usr", Size: SizeS, Time: corpusTime, Reader: bytes.NewReader(nil)})
	if err != nil {
		t.Fatalf("NewMonotonic: %v", err)
	}
	if got, err := m.Next(); !errors.Is(err, io.EOF) {
		t.Fatalf("seed: %q, %v; want an error wrapping io.EOF", got, err)
	}
	m, err = NewMonotonic(Options{Prefix: "usr", Size: SizeS, Time: corpusTime, Reader: bytes.NewReader([]byte{0, 0})})
	if err != nil {
		t.Fatalf("NewMonotonic: %v", err)
	}
	if _, err := m.Next(); err != nil {
		t.Fatalf("first: %v", err)
	}
	if got, err := m.Next(); !errors.Is(err, io.EOF) {
		t.Fatalf("step: %q, %v; want an error wrapping io.EOF", got, err)
	}
}

// A step from a Draws of the caller's own outside 1 to 2^(4w) is an error,
// not an id.
//
// sabotage: in NextWith, drop the step range check; both rows go red.
func TestMonotonicStepOutOfRange(t *testing.T) {
	for _, step := range []uint64{0, 1<<8 + 1} {
		m, err := NewMonotonic(Options{Prefix: "usr", Size: SizeS, Time: corpusTime})
		if err != nil {
			t.Fatalf("NewMonotonic: %v", err)
		}
		draws := &replayDraws{seeds: []uint64{0, 0}, steps: []uint64{step}}
		if _, err := m.NextWith(draws, time.Time{}); err != nil {
			t.Fatalf("first: %v", err)
		}
		want := fmt.Sprintf("uxid: monotonic step %d is outside 1 to 2^8", step)
		if got, err := m.NextWith(draws, time.Time{}); err == nil || err.Error() != want {
			t.Errorf("step %d: %q, %v; want %q", step, got, err, want)
		}
	}
}

// Next reads the system clock when the Options set no Time, and its ids
// carry that time; the call's own time wins over the Options' Time.
//
// sabotage: in NextWith, use time.UnixMilli(0) in place of time.Now(); the
// clock row goes red.
func TestMonotonicClock(t *testing.T) {
	m, err := NewMonotonic(Options{Prefix: "usr"})
	if err != nil {
		t.Fatalf("NewMonotonic: %v", err)
	}
	before := time.Now().UnixMilli()
	first, err := m.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	after := time.Now().UnixMilli()
	if at := readStandardTime(t, first[4:14]); at < before || at > after {
		t.Fatalf("clock: %q carries %d, outside %d to %d", first, at, before, after)
	}
	second, err := m.Next()
	if err != nil || second <= first || len(second) != len(first) {
		t.Fatalf("second: %q after %q, %v", second, first, err)
	}

	m, err = NewMonotonic(Options{Prefix: "usr", Size: SizeS, Time: corpusTime})
	if err != nil {
		t.Fatalf("NewMonotonic: %v", err)
	}
	got, err := m.NextWith(&replayDraws{seeds: []uint64{0}}, corpusTime.Add(time.Millisecond))
	if want := "usr_01hf7yat010000"; err != nil || got != want {
		t.Fatalf("the call's time: %q, %v; want %q", got, err, want)
	}
}

// The body follows Case and the delimiter is written between the prefix
// and the body, as Generate does.
//
// sabotage: in NewMonotonic, drop the upper-case alphabet; the test goes
// red.
func TestMonotonicCaseAndDelimiter(t *testing.T) {
	m, err := NewMonotonic(Options{Prefix: "Usr", Size: SizeS, Case: CaseUpper, Delimiter: "--", Time: corpusTime})
	if err != nil {
		t.Fatalf("NewMonotonic: %v", err)
	}
	got, err := m.NextWith(&replayDraws{seeds: []uint64{0xffff}}, time.Time{})
	if want := "Usr--01HF7YAT00ZZZ1"; err != nil || got != want {
		t.Fatalf("%q, %v; want %q", got, err, want)
	}
}

// A refusal after an overflow onto the compact guard leaves the last id's
// time and value as they were, so a later call whose step does not
// overflow returns an id in the last id's millisecond, after the last id.
//
// sabotage: in NextWith, on an overflow store the next millisecond and the
// new seed as the last id's before the compact guard runs; the call with
// the smaller step then lands on that stored guard millisecond, the compact
// guard refuses it and the test goes red.
func TestMonotonicRefusalKeepsTheLastId(t *testing.T) {
	m, err := NewMonotonic(Options{Prefix: "usr", Size: SizeS, CompactTime: true})
	if err != nil {
		t.Fatalf("NewMonotonic: %v", err)
	}
	before := time.UnixMilli(31<<35 - 1)
	draws := &replayDraws{seeds: []uint64{1<<24 - 100, 0}, steps: []uint64{200, 50}}
	first, err := m.NextWith(draws, before)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	if got, err := m.NextWith(draws, before); err != ErrCompactTimeOutOfRange {
		t.Fatalf("overflow onto the guard: %q, %v; want %v", got, err, ErrCompactTimeOutOfRange)
	}
	got, err := m.NextWith(draws, before)
	if want := "usr_" + first[4:12] + "zzzwe"; err != nil || got != want || got <= first {
		t.Fatalf("smaller step: %q, %v; want %q after %q", got, err, want, first)
	}
}

// addStep adds with a carry across bytes and reports an overflow of the
// field only.
//
// sabotage: in addStep, drop sum>>8 from the carry; the carry row goes
// red.
func TestAddStep(t *testing.T) {
	rows := []struct {
		field    []byte
		step     uint64
		want     []byte
		overflow bool
	}{
		{[]byte{0x00, 0xff, 0xff}, 1, []byte{0x01, 0x00, 0x00}, false},
		{[]byte{0xff, 0xff}, 1, []byte{0x00, 0x00}, true},
		{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x01}, 1 << 40, []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x01}, false},
		{[]byte{0x01}, 1 << 4, []byte{0x11}, false},
	}
	for _, r := range rows {
		field := slices.Clone(r.field)
		if overflow := addStep(field, r.step); overflow != r.overflow || !bytes.Equal(field, r.want) {
			t.Errorf("%x + %d = %x, %v; want %x, %v", r.field, r.step, field, overflow, r.want, r.overflow)
		}
	}
}
