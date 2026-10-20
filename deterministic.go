package uxid

import (
	"crypto/sha256"
	"strings"
)

// hashChars is the number of hash characters a deterministic body carries
// at the size: its standard body length less the marker. Generate checks
// the size before it asks, so SizeXL's 25 is also what any other value
// reads as.
func (s Size) hashChars() int {
	switch s {
	case SizeXS:
		return 9
	case SizeS:
		return 13
	case SizeM:
		return 17
	case SizeL:
		return 21
	}
	return 25
}

// writeDeterministic writes the deterministic body of from under prefix:
// the marker in the alphabet's case, then the first count x 5 bits of
// SHA-256 over prefix, a zero byte, from, a zero byte and one byte holding
// count. Case is not part of the hash input.
func writeDeterministic(out *strings.Builder, prefix, from string, count int, alphabet string) {
	input := make([]byte, 0, len(prefix)+len(from)+3)
	input = append(input, prefix...)
	input = append(input, 0)
	input = append(input, from...)
	input = append(input, 0, byte(count))
	digest := sha256.Sum256(input)
	out.WriteByte(alphabet[marker])
	writeLeadingBits(out, digest[:], count, alphabet)
}
