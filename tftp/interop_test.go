// Copyright 2016 Google Inc.
// Copyright 2024 Kairos contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tftp

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

var testFile = strings.Repeat(`This is a test file.

My, what a pretty test file.

I wonder if TFTP clients will be able to retrieve it!
`, 100)

func TestInterop(t *testing.T) {
	fmt.Println(len(testFile))
	prog, err := exec.LookPath("atftp")
	if err != nil {
		if e, ok := err.(*exec.Error); ok && e.Err == exec.ErrNotFound {
			t.Skip("atftp is not installed")
		}
		t.Fatalf("Error while looking for atftp: %s", err)
	}

	f, err := ioutil.TempFile("", "interop_test")
	if err != nil {
		t.Fatalf("creating temporary file: %s", err)
	}
	os.Remove(f.Name())
	defer f.Close()

	servers := []*Server{
		{
			Handler:     ConstantHandler([]byte(testFile)),
			InfoLog:     infoLog,
			TransferLog: transferLog,
		},
		{
			Handler:     ConstantHandler([]byte(testFile)),
			InfoLog:     infoLog,
			TransferLog: transferLog,
			// This Server clamps to a smaller block size.
			MaxBlockSize: 500,
		},
		{
			Handler:     ConstantHandler([]byte(testFile)),
			InfoLog:     infoLog,
			TransferLog: transferLog,
			// Lower block size to send more packets
			MaxBlockSize: 500,
			WriteTimeout: 10 * time.Millisecond,
		},
	}

	for _, s := range servers {
		fmt.Fprintf(os.Stderr, "\nUsing server: %#v\n", s)
		l, port := mkListener(t)
		defer l.Close()
		go s.Serve(l)

		options := [][]string{
			{"blksize 8"},
			{"blksize 4000"},
			{"tsize enable"},
			{"tsize enable", "blksize 1000"},
		}

		for _, opts := range options {
			c := exec.Command(prog, "--get", "--trace", "--verbose", "-r", "foo", "-l", f.Name())
			for _, o := range opts {
				c.Args = append(c.Args, "--option", o)
			}
			c.Args = append(c.Args, "127.0.0.1", strconv.Itoa(port))
			fmt.Fprintf(os.Stderr, "Fetching with: %#v\n", c.Args)

			out, err := c.CombinedOutput()
			if err != nil {
				t.Fatalf("TFTP fetch failed, command output:\n%s\n", string(out))
			}
			bs, err := ioutil.ReadFile(f.Name())
			if err != nil {
				t.Fatalf("Reading back fetched file: %s", err)
			}
			if string(bs) != testFile {
				t.Fatal("File fetched over TFTP doesn't match file served")
			}
			if err := os.Remove(f.Name()); err != nil {
				t.Fatalf("Failed to remove temp file: %s", err)
			}
		}
	}
}

func infoLog(m string) {
	fmt.Fprintf(os.Stderr, "TFTP server log: %s\n", m)
}

func transferLog(a net.Addr, p string, e error) {
	extra := ""
	if e != nil {
		extra = "(" + e.Error() + ")"
	}
	fmt.Fprintf(os.Stderr, "TFTP server transferred %q to %s %s\n", p, a, extra)
}

func mkListener(t *testing.T) (net.PacketConn, int) {
	l, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("creating listener for test: %s", err)
	}
	return l, l.LocalAddr().(*net.UDPAddr).Port
}
