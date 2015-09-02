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

// Package devo implements TiVo file decryption.  It supports both mpeg-ps and
// mpeg-ts input formats.  Decrypting requires the correct media access key
// (MAK) from the source TiVo device. DeVo is intended for personal/educational
// use only.  Decrypted files must NOT be distributed.  Piracy is not condoned!
package devo

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
)

const tsType = 0x20

type fileHeader struct {
	Magic        [4]byte
	_            [2]byte
	Flags        uint16
	_            [2]byte
	VideoOffset  uint32
	MetaSegments uint16
}

type metaHeader struct {
	ChunkSize uint32
	DataSize  uint32
	ID        uint16
	Type      uint16
}

type metadata struct {
	Header  metaHeader
	Content []byte
}

// Decrypt a TiVo file from src using the specified media access key (mak).
// The decrypted content is written to dst.
func Decrypt(dst io.Writer, src io.Reader, mak string) error {
	header, meta, err := readFileMetadata(src)
	if err != nil {
		return fmt.Errorf("devo: error parsing metadata: %s", err)
	}

	// The first metadata segment is used in entirety as an initialization vector
	iv := meta[0].Content

	srcbuf := bufio.NewReader(src)
	dstbuf := bufio.NewWriter(dst)

	if header.Flags&tsType != 0 {
		err = newTSDecryptor(mak, iv).decrypt(dstbuf, srcbuf)
	} else {
		err = newPSDecryptor(mak, iv).decrypt(dstbuf, srcbuf)
	}
	if err != nil {
		seeker, ok := src.(io.Seeker)
		if ok {
			pos, _ := seeker.Seek(0, 1)
			err = fmt.Errorf("devo: error processing input at offset 0x%08x: %s", pos, err)
		} else {
			err = fmt.Errorf("devo: error processing input: %s", err)
		}
		return err
	}
	return dstbuf.Flush()
}

func readFileMetadata(src io.Reader) (header fileHeader, meta []metadata, err error) {
	var position int64

	err = binary.Read(src, binary.BigEndian, &header)
	if err != nil {
		return
	}
	if string(header.Magic[:]) != "TiVo" {
		err = fmt.Errorf("not a tivo file (missing magic 'TiVo' marker)")
		return
	}
	position += 16 // Size of file header

	// We skip the last metadata segment because it's redundant and sometimes has bogus header data
	meta = make([]metadata, header.MetaSegments-1)
	for i := range meta {
		current := metadata{}
		err = binary.Read(src, binary.BigEndian, &current.Header)
		if err != nil {
			return
		}

		current.Content = make([]byte, current.Header.DataSize)
		_, err = src.Read(current.Content)
		if err != nil {
			return
		}
		meta[i] = current

		var endMarker uint32
		err = binary.Read(src, binary.BigEndian, &endMarker)
		if err != nil {
			return
		}
		if endMarker != 0x000000 {
			err = fmt.Errorf("metadata offset error")
			return
		}
		position += 16 + int64(current.Header.DataSize) // Size of current chunk
	}

	// Skip forward to video content
	_, err = io.CopyN(ioutil.Discard, src, int64(header.VideoOffset)-position)
	return
}
