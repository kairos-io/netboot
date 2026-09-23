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

package dhcp4

import (
	"net"
	"testing"
	"time"
)

// scriptedConn replays a fixed list of Recv outcomes, so a test can put a
// malformed datagram on the wire ahead of a good one without needing a raw
// socket (which needs root).
type scriptedConn struct {
	steps []recvStep
	pos   int
}

type recvStep struct {
	payload []byte
	err     error
}

func (c *scriptedConn) Recv(b []byte) ([]byte, *net.UDPAddr, int, error) {
	if c.pos >= len(c.steps) {
		return nil, nil, 0, net.ErrClosed
	}
	step := c.steps[c.pos]
	c.pos++
	if step.err != nil {
		return nil, nil, 0, step.err
	}
	n := copy(b, step.payload)
	// ifidx 1 is the loopback interface on Linux, so InterfaceByIndex resolves.
	return b[:n], &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 68}, 1, nil
}

func (c *scriptedConn) Send([]byte, *net.UDPAddr, int) error { return nil }
func (c *scriptedConn) Close() error                         { return nil }
func (c *scriptedConn) SetReadDeadline(time.Time) error      { return nil }
func (c *scriptedConn) SetWriteDeadline(time.Time) error     { return nil }

func discoverPacket(t *testing.T) []byte {
	t.Helper()
	mac, err := net.ParseMAC("ce:e7:7b:ef:45:f7")
	if err != nil {
		t.Fatal(err)
	}
	bs, err := (&Packet{
		Type:          MsgDiscover,
		TransactionID: []byte("1234"),
		Broadcast:     true,
		HardwareAddr:  mac,
	}).Marshal()
	if err != nil {
		t.Fatalf("marshaling packet: %s", err)
	}
	return bs
}

// A datagram the snooper conn cannot even read a UDP header out of must not
// end the receive loop: the next real boot request has to still be served.
func TestRecvDHCPSkipsAMalformedDatagram(t *testing.T) {
	good := discoverPacket(t)
	c := &Conn{
		conn: &scriptedConn{steps: []recvStep{
			{err: errMalformedPacket},
			{payload: good},
		}},
	}

	pkt, intf, err := c.RecvDHCP()
	if err != nil {
		t.Fatalf("RecvDHCP gave up after a malformed datagram: %s", err)
	}
	if pkt == nil || intf == nil {
		t.Fatalf("RecvDHCP returned no packet or no interface")
	}
	if pkt.Type != MsgDiscover {
		t.Fatalf("got packet type %s, want %s", pkt.Type, MsgDiscover)
	}
}

// A real socket error still has to be fatal, otherwise the receive loop spins
// forever on a closed connection.
func TestRecvDHCPStillFailsOnASocketError(t *testing.T) {
	c := &Conn{conn: &scriptedConn{steps: []recvStep{{err: net.ErrClosed}}}}

	if _, _, err := c.RecvDHCP(); err == nil {
		t.Fatal("RecvDHCP returned no error on a closed socket")
	}
}
