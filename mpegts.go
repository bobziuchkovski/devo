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
	"fmt"
	"github.com/bobziuchkovski/turing"
	"io"
)

const (
	tsSync          = 0x47
	tsPatID         = 0x0000
	tsIDMask        = 0x1fff
	tsPrivateType   = 0x97
	tsPrivateLength = 20
)

type packetID uint16

type tsDecryptor struct {
	pool      *cipherPool
	ciphers   map[packetID]*turing.Cipher
	pmtID     packetID
	privateID packetID
}

func newTSDecryptor(mak string, iv []byte) *tsDecryptor {
	return &tsDecryptor{
		pool:    newCipherPool(mak, iv),
		ciphers: make(map[packetID]*turing.Cipher),
	}
}

func (dec *tsDecryptor) decrypt(dst io.Writer, src *bufio.Reader) error {
	var (
		packet *tsPacket
		count  int
		err    error
	)

	for {
		count++

		// An mpeg-ts stream ends when there are no more packets to process.
		_, err = src.Peek(1)
		if err == io.EOF {
			if dec.pmtID == 0 || dec.privateID == 0 {
				err = io.ErrUnexpectedEOF
			} else {
				// Stream ends cleanly
				err = nil
			}
			break
		}
		packet, err = readTSPacket(src)
		if err != nil {
			break
		}
		err = dec.processPacket(packet)
		if err != nil {
			break
		}
		err = writeTSPacket(dst, packet)
		if err != nil {
			break
		}
	}

	if err != nil {
		err = fmt.Errorf("failed while processing mpegts packet %d: %s", count, err)
	}
	return err
}

func (dec *tsDecryptor) processPacket(packet *tsPacket) error {
	switch packet.id() {
	case tsPatID:
		return dec.processPAT(packet)
	case dec.pmtID:
		return dec.processPMT(packet)
	case dec.privateID:
		return dec.processPrivate(packet)
	default:
		if packet.scramble() != 0 {
			return dec.decryptPacket(packet)
		}
		return nil
	}
}

func (dec *tsDecryptor) processPAT(p *tsPacket) error {
	offset := 0
	payload := p.payload()
	pointer := payload[offset]
	if pointer != 0x00 {
		return fmt.Errorf("PAT table contains unsupported pointer to additional data")
	}
	offset++

	// Skip table id
	offset++

	// We expect only a single program in the stream, which should mean a length of 13
	length := joinShort(payload[offset:offset+2]) & 0x0fff
	if length != 13 {
		return fmt.Errorf("bogus PAT length -- additional programs present?")
	}
	offset += 2

	// Skip transport stream id (uint16), flags (byte), and section number/last section number info (2x uint8)
	offset += 5

	// What's remaining should be a tuple of [program id (uint16), packet id of program map (uint16)]
	offset += 2
	dec.pmtID = extractPacketID(payload[offset : offset+2])
	return nil
}

func (dec *tsDecryptor) processPMT(p *tsPacket) error {
	offset := 0
	payload := p.payload()
	pointer := payload[offset]
	if pointer != 0x00 {
		return fmt.Errorf("PMT table contains unsupported pointer to additional data")
	}
	offset++

	// Skip table id
	offset++

	// Grab table length
	length := joinShort(payload[offset:offset+2]) & 0x0fff
	offset += 2

	// Skip program id (uint16), flags (byte), section number/last section number info (2x uint8),
	// PCR PID (uint16), and Program info length (uint16)
	offset += 9

	// What's remaining should be tuples of [type (byte), pid (uint16), ES info len (uint16)]
	// offset < length only holds because there's 4 bytes prior to the table start (not included in length)
	// and 4 bytes of CRC we want to skip (more correct would be offset - 4 < length - 4)
	for uint16(offset) < length {
		streamType := payload[offset]
		if streamType == tsPrivateType {
			dec.privateID = extractPacketID(payload[offset+1 : offset+3])
			return nil
		}
		offset += 5
	}
	return fmt.Errorf("Failed to locate PID of private data")
}

func (dec *tsDecryptor) processPrivate(p *tsPacket) error {
	payload := p.payload()
	offset := 0

	if string(payload[offset:4]) != "TiVo" {
		return fmt.Errorf("bogus private data packet -- missing 'TiVo' magic bytes")
	}
	offset += 4

	// Skip 5 unused bytes
	offset += 5

	// Grab length of confounder table
	tableLength := payload[offset]
	if tableLength%tsPrivateLength != 0 {
		return fmt.Errorf("bogus private table length: %d", tableLength)
	}
	offset++

	// Extract confounders from table and construct the appropriate cipher
	dec.ciphers = make(map[packetID]*turing.Cipher)
	for i := 0; i < int(tableLength/tsPrivateLength); i++ {
		pid := extractPacketID(payload[offset : offset+2])
		streamID := payload[offset+2]
		cipher := dec.pool.getCipher(streamID, confounder(payload[offset+5:offset+9]))
		dec.ciphers[pid] = cipher
		offset += tsPrivateLength
	}

	return nil
}

func (dec *tsDecryptor) decryptPacket(p *tsPacket) error {
	pid := p.id()
	c, present := dec.ciphers[pid]
	if !present {
		return fmt.Errorf("cipher missing for scrambled packet with id 0x%04x", pid)
	}

	payload := p.payload()
	if p.payloadStart() && (joinWord(payload[0:4])>>8) == psPrefix {
		offset := 0

		// Skip PES start code, length, and flags
		offset += 8

		// Skip past remaining header length
		hdrlen := payload[offset]
		offset++
		offset += int(hdrlen)

		// Skip sequence headers/extensions
		for joinWord(payload[offset:offset+4]) == psCode(psSequenceHeader) {
			intrabyte := payload[offset+11]
			offset += 12

			// Skip Q matrices
			if intrabyte&(1<<1) != 0 {
				offset += 64
			}
			if intrabyte&(1<<0) != 0 {
				offset += 64
			}

			// Skip sequence extension
			if joinWord(payload[offset:offset+4]) == psCode(psSequenceExtension) {
				offset += 10
			}
		}

		// Skip group header
		if joinWord(payload[offset:offset+4]) == psCode(psGroupHeader) {
			offset += 8
		}

		payload = payload[offset:]
	}

	c.XORKeyStream(payload, payload)
	p.clearScramble()
	return nil
}

type tsPacket struct {
	content [188]byte
}

func readTSPacket(src io.Reader) (packet *tsPacket, err error) {
	packet = &tsPacket{}
	_, err = io.ReadFull(src, packet.content[:])
	if err != nil {
		return
	}

	if packet.content[0] != tsSync {
		err = fmt.Errorf("expected sync byte, got 0x%02x instead", packet.content[0])
	}
	return
}

func writeTSPacket(dst io.Writer, packet *tsPacket) error {
	_, err := dst.Write(packet.content[:])
	return err
}

func (p *tsPacket) payloadStart() bool {
	return p.content[1]&(1<<6) != 0
}

func (p *tsPacket) id() packetID {
	return extractPacketID(p.content[1:3])
}

func (p *tsPacket) scramble() uint8 {
	return p.content[3] >> 6
}

func (p *tsPacket) clearScramble() {
	p.content[3] = p.content[3] ^ 0xc0
}

func (p *tsPacket) hasAdaptation() bool {
	return p.content[3]&(1<<5) != 0
}

func (p *tsPacket) hasPayload() bool {
	return p.content[3]&(1<<4) != 0
}

func (p *tsPacket) counter() uint8 {
	return p.content[3] & 0x0f
}

func (p *tsPacket) payload() []byte {
	var offset uint8
	if p.hasAdaptation() {
		offset = 5 + p.content[4]
	} else {
		offset = 4
	}
	return p.content[offset:]
}

func extractPacketID(b []byte) packetID {
	return packetID(joinShort(b) & tsIDMask)
}
