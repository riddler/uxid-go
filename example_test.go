package uxid_test

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/riddler/uxid-go"
)

// The README's basic usage: a prefixed id, a reproduced vector, a
// deterministic id, Decode and the checks, and a monotonic generator. A test
// keeps this body identical to the README block.
func Example() {
	// A prefix names the resource; the body is a time and randomness.
	id, err := uxid.Generate(uxid.Options{Prefix: "usr"})
	if err != nil {
		panic(err)
	}
	fmt.Println(strings.HasPrefix(id, "usr_"), len(id)) // true 30

	// A size shrinks the random part for low-cardinality resources, and a fixed
	// time and fixed random bytes reproduce an id exactly. Fixed bytes are for
	// test vectors only, never a source of randomness.
	short, err := uxid.Generate(uxid.Options{
		Prefix: "cus",
		Size:   uxid.SizeS,
		Time:   time.UnixMilli(1_700_000_000_000),
		Rand:   []byte{0x01, 0x23},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(short) // cus_01hf7yat0004h1

	// Deterministic: the same prefix and input give the same id, every time.
	named, err := uxid.Generate(uxid.Options{Prefix: "usr", From: new("alice@example.com")})
	if err != nil {
		panic(err)
	}
	fmt.Println(named, uxid.IsDeterministic(named, "_")) // usr_zcvt7epac0t1ebcsjfyf7cwz25 true

	// Read an id back. A malformed id is a *DecodeError, matched with errors.As.
	decoded, err := uxid.Decode(short)
	if err != nil {
		panic(err)
	}
	fmt.Println(decoded.Prefix, decoded.Size, decoded.Time.UnixMilli())           // cus s 1700000000000
	fmt.Println(uxid.IsValid(short, "_", "cus"), uxid.IsValid(short, "_", "usr")) // true false
	_, err = uxid.Decode("cus_")
	fmt.Println(err) // empty

	// Monotonic: ids from one generator increase strictly, also within one
	// millisecond. You hold the generator; the module keeps no global state.
	ids, err := uxid.NewMonotonic(uxid.Options{Prefix: "usr", Size: uxid.SizeM})
	if err != nil {
		panic(err)
	}
	first, err := ids.Next()
	if err != nil {
		panic(err)
	}
	second, err := ids.Next()
	if err != nil {
		panic(err)
	}
	fmt.Println(first < second) // true
	// Output:
	// true 30
	// cus_01hf7yat0004h1
	// usr_zcvt7epac0t1ebcsjfyf7cwz25 true
	// cus s 1700000000000
	// true false
	// empty
	// true
}

// An id with the defaults but a prefix: size xl, standard time, lower case,
// the current time and random bytes from crypto/rand.Reader.
func ExampleGenerate() {
	id, err := uxid.Generate(uxid.Options{Prefix: "usr"})
	if err != nil {
		panic(err)
	}
	fmt.Println(len(id))
	// Output: 30
}

// Fixed random bytes and a fixed time reproduce a test vector. Fixed bytes
// are for vectors only, never a source of randomness.
func ExampleGenerate_vector() {
	id, err := uxid.Generate(uxid.Options{
		Prefix: "cus",
		Size:   uxid.SizeS,
		Time:   time.UnixMilli(1_700_000_000_000),
		Rand:   []byte{0x01, 0x23},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(id)
	// Output: cus_01hf7yat0004h1
}

// A refusal is a sentinel error, matched with errors.Is; its text is the
// name the conformance corpus gives it.
func ExampleGenerate_refusal() {
	_, err := uxid.Generate(uxid.Options{Prefix: "usr", RandSize: new(11)})
	fmt.Println(errors.Is(err, uxid.ErrRandSizeTooLarge), err)
	// Output: true rand_size_too_large
}

// A deterministic id: the same prefix, string and size give the same id on
// every call, on every machine.
func ExampleGenerate_deterministic() {
	id, err := uxid.Generate(uxid.Options{Prefix: "usr", From: new("alice@example.com")})
	if err != nil {
		panic(err)
	}
	fmt.Println(id)
	// Output: usr_zcvt7epac0t1ebcsjfyf7cwz25
}

// A monotonic generator's ids increase strictly, also within one
// millisecond. The generator is safe for concurrent use, so one may be
// shared by every goroutine that makes ids of its prefix and size.
func ExampleMonotonic() {
	ids, err := uxid.NewMonotonic(uxid.Options{Prefix: "usr", Size: uxid.SizeM})
	if err != nil {
		panic(err)
	}
	first, err := ids.Next()
	if err != nil {
		panic(err)
	}
	second, err := ids.Next()
	if err != nil {
		panic(err)
	}
	fmt.Println(len(first), first < second)
	// Output: 22 true
}

// With a fixed time and a byte source of zeros, the first id seeds the
// random field and the next, in the same millisecond, steps it. A fixed
// byte source is for tests only, never for real ids.
func ExampleMonotonic_NextWith() {
	ids, err := uxid.NewMonotonic(uxid.Options{
		Prefix: "usr",
		Size:   uxid.SizeS,
		Reader: bytes.NewReader(make([]byte, 3)),
	})
	if err != nil {
		panic(err)
	}
	at := time.UnixMilli(1_700_000_000_000)
	for range 2 {
		id, err := ids.NextWith(nil, at)
		if err != nil {
			panic(err)
		}
		fmt.Println(id)
	}
	// Output:
	// usr_01hf7yat000000
	// usr_01hf7yat000001
}

// A monotonic generator refuses a deterministic body and fixed random
// bytes when it is built.
func ExampleNewMonotonic_refusal() {
	_, err := uxid.NewMonotonic(uxid.Options{Prefix: "usr", From: new("alice@example.com")})
	fmt.Println(errors.Is(err, uxid.ErrFromWithMonotonic), err)
	// Output: true from_with_monotonic
}

// Decode reads an id back: its prefix, size, time form, and time. A
// refusal is a *DecodeError, matched with errors.As; its text is the
// corpus name of the reason.
func ExampleDecode() {
	decoded, err := uxid.Decode("cus_01hf7yat0004h1")
	if err != nil {
		panic(err)
	}
	fmt.Println(decoded.Prefix, decoded.Size, decoded.Compact, decoded.Time.UnixMilli())

	_, err = uxid.Decode("cus_01hf7yat000")
	var decodeErr *uxid.DecodeError
	fmt.Println(errors.As(err, &decodeErr), decodeErr.Reason == uxid.ReasonInvalidLength, err)
	// Output:
	// cus s false 1700000000000
	// true true invalid_length
}

// A compact time carries only its low 40 bits; the clock in DecodeOptions
// supplies the rest, so a test can fix it.
func ExampleDecodeWith() {
	decoded, err := uxid.DecodeWith("cus--hf7yat0004hm5", uxid.DecodeOptions{
		Delimiter: "--",
		Now:       time.UnixMilli(1_700_000_000_000),
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(decoded.Prefix, decoded.Size, decoded.Compact, decoded.Time.UnixMilli())
	// Output: cus s true 1700000000000
}

func ExampleIsValid() {
	fmt.Println(uxid.IsValid("cus_01hf7yat0004h1", "_", ""))
	fmt.Println(uxid.IsValid("cus_01hf7yat0004h1", "_", "usr"))
	fmt.Println(uxid.IsValid("cus_01hf7yat0004h12", "_", ""))
	// Output:
	// true
	// false
	// false
}

func ExampleIsDeterministic() {
	fmt.Println(uxid.IsDeterministic("usr_zcvt7epac0t1ebcsjfyf7cwz25", "_"))
	fmt.Println(uxid.IsDeterministic("usr_01hf7yat0004h1", "_"))
	// Output:
	// true
	// false
}

func ExampleParseSize() {
	size, err := uxid.ParseSize("medium")
	if err != nil {
		panic(err)
	}
	fmt.Println(size)
	// Output: m
}

// The README's basic-usage block and Example are one text: the block is the
// body of Example, less its Output comment, with one tab of indentation
// removed.
//
// sabotage: change a character in either the README block or Example's body and this test goes red.
func TestReadmeBlockIsExample(t *testing.T) {
	source, err := os.ReadFile("example_test.go")
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.ParseFile(token.NewFileSet(), "example_test.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	var fixture string
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "Example" {
			continue
		}
		// Positions are 1-based offsets into the file; the body is the text
		// between the braces.
		fixture = string(source[fn.Body.Lbrace : fn.Body.Rbrace-1])
	}
	if fixture == "" {
		t.Fatal("no func Example found")
	}
	body, _, found := strings.Cut(fixture, "\n\t// Output:")
	if !found {
		t.Fatal("Example has no Output comment")
	}
	var want strings.Builder
	for _, line := range strings.SplitAfter(strings.TrimPrefix(body, "\n"), "\n") {
		want.WriteString(strings.TrimPrefix(line, "\t"))
	}

	readme, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	_, rest, found := strings.Cut(string(readme), "```go\n")
	if !found {
		t.Fatal("README has no go block")
	}
	block, _, found := strings.Cut(rest, "```")
	if !found {
		t.Fatal("README's go block does not close")
	}
	if got := strings.TrimSuffix(block, "\n"); got != strings.TrimSuffix(want.String(), "\n") {
		t.Errorf("README block differs from Example's body\nREADME:\n%s\nExample:\n%s", got, want.String())
	}
}
