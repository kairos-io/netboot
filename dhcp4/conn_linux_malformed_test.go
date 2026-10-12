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

//go:build linux

package dhcp4

import (
	"errors"
	"testing"
)

// The BPF filter on the snooper socket only reads the destination port, so a
// datagram that stops inside its own UDP header reaches Recv. It has to be
// reported as malformed, because RecvDHCP treats anything else as the socket
// dying.
func TestUDPPayloadReportsAShortDatagramAsMalformed(t *testing.T) {
	for _, n := range []int{0, 4, 7} {
		_, _, err := udpPayload(make([]byte, n))
		if !errors.Is(err, errMalformedPacket) {
			t.Errorf("udpPayload on a %d byte datagram gave %v, want errMalformedPacket", n, err)
		}
	}
}

func TestUDPPayloadSplitsAWellFormedDatagram(t *testing.T) {
	// src port 68, dst port 67, length 9, no checksum, one byte of payload.
	payload, sport, err := udpPayload([]byte{0, 68, 0, 67, 0, 9, 0, 0, 0x42})
	if err != nil {
		t.Fatalf("udpPayload on a well-formed datagram: %s", err)
	}
	if sport != 68 {
		t.Errorf("source port %d, want 68", sport)
	}
	if len(payload) != 1 || payload[0] != 0x42 {
		t.Errorf("payload %v, want [66]", payload)
	}
}
