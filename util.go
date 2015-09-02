// Copyright (c) 2016 Bob Ziuchkovski
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package devo

func confounder(scrambled []byte) (confounder [3]byte) {
	if len(scrambled) < 4 {
		panic("expected at least 4 bytes")
	}

	// This just shifts bits.  The octets end up as 00000011 11111122 22222333
	confounder[0] = ((scrambled[0] & 0x3f) << 2) | (scrambled[1] >> 6)
	confounder[1] = ((scrambled[1] & 0x3f) << 2) | (scrambled[2] >> 6)
	confounder[2] = ((scrambled[2] & 0x1f) << 3) | (scrambled[3] >> 5)
	return
}

func joinShort(octets []byte) uint16 {
	if len(octets) != 2 {
		panic("expected 2 bytes")
	}
	return (uint16(octets[0]) << 8) | uint16(octets[1])
}

func joinWord(octets []byte) uint32 {
	if len(octets) != 4 {
		panic("expected 4 bytes")
	}
	return (uint32(octets[0]) << 24) | (uint32(octets[1]) << 16) | (uint32(octets[2]) << 8) | uint32(octets[3])
}

func wordOctet(word uint32, n uint) byte {
	return byte((word >> (24 - n*8)) & 0xff)
}
