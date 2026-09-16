// Copyright 2026 Kairos contributors
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
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

// rrqPacket builds a read request for fname in octet mode, followed by the
// given option name/value pairs.
func rrqPacket(fname string, opts ...string) []byte {
	var b bytes.Buffer
	b.Write(binary.BigEndian.AppendUint16(nil, 1))
	b.WriteString(fname)
	b.WriteByte(0)
	b.WriteString("octet")
	b.WriteByte(0)
	for _, o := range opts {
		b.WriteString(o)
		b.WriteByte(0)
	}
	return b.Bytes()
}

func TestParseRRQBlockSize(t *testing.T) {
	tests := []struct {
		name    string
		val     string
		want    int64
		wantErr string
	}{
		{name: "smallest accepted", val: "8", want: 8},
		{name: "largest accepted", val: "65464", want: 65464},
		{name: "too small", val: "7", wantErr: "unsupported block size 7"},
		{name: "too large", val: "70000", wantErr: "unsupported block size 70000"},
		{name: "not a number", val: "big", wantErr: `non-integer block size value "big"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := parseRRQ(rrqPacket("boot.img", "blksize", tt.val))

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("parseRRQ(blksize=%s) = %v, want no error", tt.val, err)
				}
				if req.BlockSize != tt.want {
					t.Errorf("BlockSize = %d, want %d", req.BlockSize, tt.want)
				}
				return
			}

			if err == nil {
				t.Fatalf("parseRRQ(blksize=%s) succeeded, want error %q", tt.val, tt.wantErr)
			}
			// The rejected value has to reach the client as the number it
			// sent, so the message is checked in full rather than only for
			// the absence of an escape.
			if err.Error() != tt.wantErr {
				t.Errorf("error = %q, want %q", err.Error(), tt.wantErr)
			}
			if strings.Contains(err.Error(), `\u{`) {
				t.Errorf("error %q formats the block size as a rune", err.Error())
			}
		})
	}
}
