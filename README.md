[![Build Status](https://travis-ci.org/ziuchkovski/devo.svg?branch=master)](https://travis-ci.org/ziuchkovski/devo)
[![Coverage](http://gocover.io/_badge/github.com/ziuchkovski/devo?0)](http://gocover.io/github.com/ziuchkovski/devo)
[![Report Card](http://goreportcard.com/badge/ziuchkovski/devo)](http://goreportcard.com/report/ziuchkovski/devo)
[![GoDoc](https://godoc.org/github.com/ziuchkovski/devo?status.svg)](https://godoc.org/github.com/ziuchkovski/devo)

# DeVo

## Overview

DeVo decrypts TiVo files.  It supports both mpeg-ps and mpeg-ts input formats.
Decrypting requires the correct media access key (MAK) from the source TiVo device.
DeVo is intended for personal/educational use only.  Decrypted files must NOT be distributed.
Piracy is not condoned!

## Usage

`devo -m [MAK] -i [INPUT] -o [OUTPUT]`

If the output file is garbled, double-check the provided access key.
DeVo makes no attempt to detect bogus access keys.

## Library

DeVo is implemented in pure (Go)[https://golang.org/] and can be used as a library.
Please see the [godocs](https://godoc.org/github.com/ziuchkovski/devo) for details.

## Authors

Bob Ziuchkovski (@ziuchkovski)

## License (MIT)

Copyright (c) 2016 Bob Ziuchkovski

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
