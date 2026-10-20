package uxid_test

import (
	"errors"
	"fmt"
	"time"

	"github.com/riddler/uxid-go"
)

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
