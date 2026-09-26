// Copyright 2026 Kairos contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"net"
	"net/url"
	"strings"
	"testing"

	"github.com/kairos-io/netboot/types"
)

// mustMAC parses a MAC address or fails the test.
func mustMAC(t *testing.T, s string) net.HardwareAddr {
	t.Helper()
	mac, err := net.ParseMAC(s)
	if err != nil {
		t.Fatalf("parsing MAC %q: %s", s, err)
	}
	return mac
}

// chainURL pulls the URL out of the single "chain" line of an EFI iPXE script.
func chainURL(t *testing.T, script []byte) *url.URL {
	t.Helper()
	for _, line := range strings.Split(string(script), "\n") {
		if !strings.HasPrefix(line, "chain ") {
			continue
		}
		fields := strings.Fields(line)
		raw := fields[len(fields)-1]
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatalf("chain URL %q does not parse: %s", raw, err)
		}
		return u
	}
	t.Fatalf("no chain line in script:\n%s", script)
	return nil
}

// The EFI script has to survive the round trip through the URL the client
// actually fetches: handleFile reads the "name" query parameter and asks the
// Booter for exactly that ID, so whatever went into the script has to come
// back out of it unchanged. ipxeScript escapes every value it interpolates;
// ipxeScriptEfi has to do the same.
func TestIpxeScriptEfiEscapesTheName(t *testing.T) {
	mach := types.Machine{MAC: mustMAC(t, "01:02:03:04:05:06")}

	for _, efi := range []string{
		"http://example.com/boot.efi?a=1&type=kernel",
		"kairos&type=kernel",
		"boot file.efi",
		"a+b.efi",
		"frag#ment.efi",
		"100%.efi",
	} {
		t.Run(efi, func(t *testing.T) {
			spec := &types.Spec{Efi: types.ID(efi)}

			script, err := ipxeScriptEfi(mach, spec, "localhost:1234")
			if err != nil {
				t.Fatalf("building EFI script: %s", err)
			}

			u := chainURL(t, script)
			q := u.Query()

			if got := q.Get("name"); got != efi {
				t.Errorf("name round-tripped as %q, want %q\nscript: %s", got, efi, script)
			}
			if got := q.Get("type"); got != "efi" {
				t.Errorf("type round-tripped as %q, want %q\nscript: %s", got, "efi", script)
			}
			if got := q.Get("mac"); got != mach.MAC.String() {
				t.Errorf("mac round-tripped as %q, want %q\nscript: %s", got, mach.MAC.String(), script)
			}
		})
	}
}
