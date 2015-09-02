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

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	psPrefix            = 0x000001
	psSequenceHeader    = 0xb3
	psSequenceExtension = 0xb5
	psGroupHeader       = 0xb8
	psProgramEnd        = 0xb9
	psPackStart         = 0xba
	psSystemHeader      = 0xbb
	psStreamMap         = 0xbc
)

// Length of headers by flag bit (bit 0 .. 7)
var flagLengths = []int{
	1, // Extension
	2, // CRC
	1, // Additional Copyright Info
	1, // DSM Trick Mode
	3, // ES Rate
	6, // ESCR
	5, // DTS
	5, // PTS
}

type psDecryptor struct {
	pool *cipherPool
}

func newPSDecryptor(mak string, iv []byte) *psDecryptor {
	return &psDecryptor{
		pool: newCipherPool(mak, iv),
	}
}

func (dec *psDecryptor) decrypt(dst io.Writer, src *bufio.Reader) error {
	var (
		packet *psPacket
		count  int
		err    error
	)

	for {
		count++
		packet, err = readPSPacket(src)
		if err != nil {
			break
		}
		err = dec.processPacket(packet)
		if err != nil {
			break
		}
		err = writePSPacket(dst, packet)
		if err != nil {
			break
		}
		if packet.id == psProgramEnd {
			break
		}
	}

	// An EOF is unexpected.  We expect to read psProgramEnd prior to hitting EOF.
	if err == io.EOF {
		err = io.ErrUnexpectedEOF
	}
	if err != nil {
		err = fmt.Errorf("failed while processing mpegps packet %d: %s", count, err)
	}
	return err
}

func (dec *psDecryptor) processPacket(packet *psPacket) (err error) {
	switch packet.id {
	case psStreamMap:
		// Twiddle stream map for decrypted stream
		packet.payload()[0] &= 0xdf
	default:
		if packet.scramble() != 0 {
			dec.decryptPacket(packet)
		}
	}
	return
}

func (dec *psDecryptor) decryptPacket(packet *psPacket) {
	cipher := dec.pool.getCipher(packet.id, confounder(packet.privateData()[1:5]))

	// We throw out the first four bytes of the cipher stream
	// Don't ask why...this is the same thing tivodecode does
	var dummy [4]byte
	cipher.XORKeyStream(dummy[:], dummy[:])

	// Use the rest of the stream to decrypt the packet payload
	encrypted := packet.payload()
	cipher.XORKeyStream(encrypted, encrypted)
	packet.clearScramble()
}

type psPacket struct {
	id      uint8
	content []byte
}

func (packet *psPacket) scramble() uint8 {
	switch packet.id {
	case psPackStart, psProgramEnd:
		return 0
	default:
		return (packet.content[0] & 0x30) >> 4
	}
}

func (packet *psPacket) clearScramble() {
	if packet.scramble() != 0 {
		packet.content[0] &= 0xcf
	}
}

func (packet *psPacket) payload() []byte {
	switch packet.id {
	case psPackStart, psStreamMap, psSystemHeader:
		return packet.content
	default:
		hdrlen := int(packet.content[2]) + 3
		return packet.content[hdrlen:]
	}
}

func (packet *psPacket) privateData() []byte {
	flagPos, lenPos, remaining := 1, 2, 3
	flags := packet.content[flagPos]
	hdrlen := int(packet.content[lenPos]) + remaining

	// Skip PTS, DTS, ESCR, ES Rate, DSM Trick Mode, Copyright Info, CRC, and Extension data
	offset := remaining
	for bit, len := range flagLengths {
		if flags&(1<<uint(bit)) != 0 {
			offset += len
		}
	}

	// XXX We might be returning more than just the private data here...
	return packet.content[offset : hdrlen-remaining]
}

func readPSPacket(src io.Reader) (packet *psPacket, err error) {
	var code uint32
	err = binary.Read(src, binary.BigEndian, &code)
	if err != nil {
		return
	}
	if (code >> 8) != psPrefix {
		err = fmt.Errorf("invalid PS packet code: 0x%08x", code)
		return
	}
	packet = &psPacket{id: wordOctet(code, 3)}

	switch packet.id {
	case psProgramEnd:
		// Empty content
		packet.content = make([]byte, 0)
	case psPackStart:
		// Pack start content
		packet.content = make([]byte, 10)
		_, err = io.ReadFull(src, packet.content)
		if err != nil {
			return
		}

		// Stuffing bytes
		scount := packet.content[9] & 0x07
		stuffing := make([]byte, scount)
		_, err = io.ReadFull(src, stuffing)
		if err != nil {
			return
		}
		packet.content = append(packet.content, stuffing...)
	default:
		// Remaining packets are all in type-length-value format and we already have the type (code)
		var length uint16
		err = binary.Read(src, binary.BigEndian, &length)
		if err != nil {
			return
		}
		packet.content = make([]byte, length)
		_, err = io.ReadFull(src, packet.content)
		if err != nil {
			return
		}
	}
	return
}

func writePSPacket(dst io.Writer, p *psPacket) (err error) {
	var code = (psPrefix << 8) | uint32(p.id)
	err = binary.Write(dst, binary.BigEndian, code)
	if err != nil {
		return
	}
	switch p.id {
	case psPackStart, psProgramEnd:
		// Not in type-length-value format, so don't write length
	default:
		err = binary.Write(dst, binary.BigEndian, uint16(len(p.content)))
		if err != nil {
			return
		}
	}
	_, err = dst.Write(p.content)
	return
}

func psCode(id uint8) uint32 {
	return (psPrefix << 8) | uint32(id)
}
