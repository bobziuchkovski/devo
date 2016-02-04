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
	"crypto/sha1"
	"github.com/bobziuchkovski/turing"
)

type cipherHandle struct {
	id         uint8
	confounder [3]byte
}

type cipherPool struct {
	mak     string // Media access key (MAK) from TiVo unit
	iv      []byte // Initialization vector (IV) from file metadata
	ciphers map[cipherHandle]*turing.Cipher
}

func newCipherPool(mak string, iv []byte) *cipherPool {
	return &cipherPool{
		mak:     mak,
		iv:      iv,
		ciphers: make(map[cipherHandle]*turing.Cipher),
	}
}

func (pool *cipherPool) getCipher(id uint8, confounder [3]byte) *turing.Cipher {
	handle := cipherHandle{id: id, confounder: confounder}
	c, present := pool.ciphers[handle]
	if !present {
		basekey := sha1.Sum(append([]byte(pool.mak), pool.iv...))
		derivedkey := sha1.Sum(append(basekey[:16], handle.id))
		derivediv := sha1.Sum(append(basekey[:16], handle.id, handle.confounder[0], handle.confounder[1], handle.confounder[2]))

		var err error
		c, err = turing.NewCipher(derivedkey[:], derivediv[:])
		if err != nil {
			panic(err)
		}
		pool.ciphers[handle] = c
	}
	return c
}
