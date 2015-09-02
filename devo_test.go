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
	"crypto/md5"
	"os"
	"reflect"
	"testing"
)

// These are baseline sanity tests.
// The decrypted md5s might change as the result of legitimate bug fixes,
// causing the tests to fail.  It's unfortunate, but it's better than
// no tests at all.

type devoTest struct {
	name         string
	file         string
	mak          string
	decryptedMd5 []byte
}

func TestPS(t *testing.T) {
	test := devoTest{
		name: "Basic PS Video",
		file: "test.mpegps.tivo",
		mak:  "3886854575",
		decryptedMd5: []byte{
			0x19, 0x04, 0x6a, 0x41, 0x0d, 0x64, 0x9c, 0xad,
			0x5a, 0xbd, 0xfc, 0xe5, 0xad, 0xa0, 0xe9, 0xd6,
		},
	}
	runDevoTest(t, test)
}

func TestTS(t *testing.T) {
	test := devoTest{
		name: "Basic TS Video",
		file: "test.mpegts.tivo",
		mak:  "3886854575",
		decryptedMd5: []byte{
			0x3b, 0xf0, 0x9c, 0xa6, 0xad, 0xbe, 0x97, 0x14,
			0xb2, 0xac, 0x59, 0x7c, 0x80, 0xfa, 0xdf, 0xc0,
		},
	}
	runDevoTest(t, test)
}

func runDevoTest(t *testing.T, test devoTest) {
	r, err := os.Open(test.file)
	if err != nil {
		t.Errorf("Encountered unexpected error reading test file.  Test: %s, Error: %s", test.name, err)
	}
	defer r.Close()

	h := md5.New()
	err = Decrypt(h, r, test.mak)
	if err != nil {
		t.Errorf("Encountered unexpected error decrypting test file.  Test: %s, Error: %s", test.name, err)
	}

	if !reflect.DeepEqual(test.decryptedMd5, h.Sum(nil)) {
		t.Errorf("Decrypted file is invalid.  Test: %s", test.name)
	}
}
