// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package internal

import (
	"strconv"
	"unicode/utf8"
)

// A Builder is used to efficiently build a string using Write methods.
// It minimizes memory copying. The zero value is ready to use.
// Do not copy a non-zero Builder.
type StringBuilder struct {
	buf []byte
}

// Len returns the number of accumulated bytes; b.Len() == len(b.String()).
func (b *StringBuilder) Len() int { return len(b.buf) }

// Cap returns the capacity of the builder's underlying byte slice. It is the
// total space allocated for the string being built and includes any bytes
// already written.
func (b *StringBuilder) Cap() int { return cap(b.buf) }

// grow copies the buffer to a new, larger buffer so that there are at least n
// bytes of capacity beyond len(b.buf).
func (b *StringBuilder) grow(n int) {
	buf := make([]byte, len(b.buf), 2*cap(b.buf)+n)
	copy(buf, b.buf)
	b.buf = buf
}

// Grow grows b's capacity, if necessary, to guarantee space for
// another n bytes. After Grow(n), at least n bytes can be written to b
// without another allocation. If n is negative, Grow panics.
func (b *StringBuilder) Grow(n int) {
	if n < 0 {
		panic("strings.Builder.Grow: negative count")
	}
	if cap(b.buf)-len(b.buf) < n {
		b.grow(n)
	}
}

// Write appends the contents of p to b's buffer.
// Write always returns len(p), nil.
func (b *StringBuilder) Write(p []byte) (int, error) {
	b.buf = append(b.buf, p...)
	return len(p), nil
}

// WriteByte appends the byte c to b's buffer.
// The returned error is always nil.
func (b *StringBuilder) WriteByte(c byte) error {
	b.buf = append(b.buf, c)
	return nil
}

// WriteRune appends the UTF-8 encoding of Unicode code point r to b's buffer.
// It returns the length of r and a nil error.
func (b *StringBuilder) WriteRune(r rune) (int, error) {
	if r < utf8.RuneSelf {
		b.buf = append(b.buf, byte(r))
		return 1, nil
	}
	l := len(b.buf)
	if cap(b.buf)-l < utf8.UTFMax {
		b.grow(utf8.UTFMax)
	}
	n := utf8.EncodeRune(b.buf[l:l+utf8.UTFMax], r)
	b.buf = b.buf[:l+n]
	return n, nil
}

// WriteString appends the contents of s to b's buffer.
// It returns the length of s and a nil error.
func (b *StringBuilder) WriteString(s string) (int, error) {
	b.buf = append(b.buf, s...)
	return len(s), nil
}

// String returns the accumulated string in a separate copy.
func (b *StringBuilder) String() string {
	return string(b.buf)
}

// Bytes returns the accumulated bytes.
func (b *StringBuilder) Bytes() []byte {
	return b.buf
}

// Reset resets the Builder to be empty.
func (b *StringBuilder) Reset() {
	b.buf = b.buf[:0]
}

// AppendFloat appends the string form of the floating-point number f,
// as generated by FormatFloat, to dst and returns the extended buffer.
func (b *StringBuilder) AppendFloat(f float64, fmt byte, prec, bitSize int) {
	b.buf = strconv.AppendFloat(b.buf, f, fmt, prec, bitSize)
}

// AppendInt appends the string form of the integer i,
// as generated by FormatInt, to dst and returns the extended buffer.
func (b *StringBuilder) AppendInt(i int64, base int) {
	b.buf = strconv.AppendInt(b.buf, i, base)
}
