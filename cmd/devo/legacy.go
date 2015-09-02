// +build !go1.5

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

package main

import (
	"fmt"
	"github.com/ziuchkovski/devo"
	"github.com/ziuchkovski/writ"
	"io"
	"os"
	"regexp"
	"runtime"
)

const (
	usage  = "Usage: devo [OPTION]..."
	header = `
DeVo decrypts TiVo recordings.  It is intended for personal/educational use only.
Decrypted files must NOT be distributed.  Piracy is NOT condoned!`
	footer = "DeVo home page: https://github.com/ziuchkovski/devo"
)

type config struct {
	Input       io.Reader      `option:"i, input" placeholder:"FILE" description:"The encrypted input TiVo file"`
	Output      io.WriteCloser `option:"o, output" placeholder:"FILE" description:"The decrypted output video file"`
	AccessKey   string         `option:"m, mak" placeholder:"MAK" description:"The 10-digit media access key (MAK) from your TiVo"`
	HelpFlag    bool           `flag:"h, help" description:"Display this help text and exit"`
	VersionFlag bool           `flag:"version" description:"Display version information and exit"`
}

func (c config) validate() error {
	if c.Input == nil {
		return fmt.Errorf("-i/--input must be specified")
	}
	if c.Output == nil {
		return fmt.Errorf("-o/--output must be specified")
	}
	if c.AccessKey == "" {
		return fmt.Errorf("-m/--mak is required")
	}
	if !regexp.MustCompile("^\\d{10}$").MatchString(c.AccessKey) {
		return fmt.Errorf("-m/--mak must be a 10 digit value")
	}
	return nil
}

func main() {
	config := &config{}
	cmd := writ.New("devo", config)
	cmd.Help.Usage = usage
	cmd.Help.Header = header
	cmd.Help.Footer = footer
	_, positional, err := cmd.Decode(os.Args[1:])

	if err != nil || config.HelpFlag {
		cmd.ExitHelp(err)
	}
	if config.VersionFlag {
		fmt.Fprintf(os.Stdout, "DeVo version %d.%d.%d\nCompiled with %s\n", devo.Version.Major, devo.Version.Minor, devo.Version.Patch, runtime.Version())
		os.Exit(0)
	}
	if len(positional) != 0 {
		cmd.ExitHelp(fmt.Errorf("too many arguments provided"))
	}
	err = config.validate()
	if err != nil {
		cmd.ExitHelp(err)
	}

	defer config.Output.Close()
	check(devo.Decrypt(config.Output, config.Input, config.AccessKey))
}

func check(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
