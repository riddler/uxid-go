package uxid

import "strings"

// The Crockford Base32 symbols in value order, one table per case: 32
// symbols in ASCII order, with I, L, O and U left out.
const (
	lowerAlphabet = "0123456789abcdefghjkmnpqrstvwxyz"
	upperAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
)

// writeInteger writes the low count 5-bit groups of value, most significant
// group first. The first group holds whatever bits remain above the others,
// which is how the standard 48-bit time writes its 3-bit first group.
func writeInteger(out *strings.Builder, value uint64, count int, alphabet string) {
	for i := count - 1; i >= 0; i-- {
		out.WriteByte(alphabet[(value>>(5*uint(i)))&31])
	}
}

// writeLeadingBits writes the first count 5-bit groups of bytes, most
// significant bit first, with no short group. bytes holds at least
// count x 5 bits.
func writeLeadingBits(out *strings.Builder, bytes []byte, count int, alphabet string) {
	var buffer uint16
	var bits uint
	for _, b := range bytes {
		buffer = buffer<<8 | uint16(b)
		bits += 8
		for bits >= 5 {
			if count == 0 {
				return
			}
			bits -= 5
			out.WriteByte(alphabet[(buffer>>bits)&31])
			count--
		}
		buffer &= 1<<bits - 1
	}
}

// writeBytes writes bytes as 5-bit groups, most significant bit first. When
// the bit count is not a multiple of 5 the last group is short, and it is
// written as its raw value rather than shifted or padded: the two bytes
// ff ff are zzz1.
func writeBytes(out *strings.Builder, bytes []byte, alphabet string) {
	var buffer uint16
	var bits uint
	for _, b := range bytes {
		buffer = buffer<<8 | uint16(b)
		bits += 8
		for bits >= 5 {
			bits -= 5
			out.WriteByte(alphabet[(buffer>>bits)&31])
		}
		buffer &= 1<<bits - 1
	}
	if bits > 0 {
		out.WriteByte(alphabet[buffer])
	}
}
