// Copyright 2024 Kairos contributors
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

package booters

import (
	"testing"

	"github.com/kairos-io/netboot/types"
)

func TestFileNameFromLocation(t *testing.T) {
	for _, tc := range []struct {
		name     string
		location string
		want     string
	}{
		{"local path", "/build/qa-alpha.sysext.raw", "qa-alpha.sysext.raw"},
		{"bare name", "qa-alpha.sysext.raw", "qa-alpha.sysext.raw"},
		{"http url", "http://example.com/img/qa-alpha.sysext.raw", "qa-alpha.sysext.raw"},
		// A query must not end up in the name: a sysext is only merged
		// when its file name ends in .raw, so "a.raw?token=x" is useless.
		{"http url with query", "http://example.com/a.raw?token=secret", "a.raw"},
		{"percent escaped", "http://example.com/qa%20alpha.raw", "qa alpha.raw"},
		{"url with no path", "http://example.com", ""},
		{"url root only", "http://example.com/", ""},
		{"empty", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := fileNameFromLocation(tc.location); got != tc.want {
				t.Errorf("fileNameFromLocation(%q) = %q, want %q", tc.location, got, tc.want)
			}
		})
	}
}

// Every file a cmdline hands to the booted system is served under its own
// name, so the system can tell two of them apart. See kairos-io/kairos#5369.
func TestStaticBooterNamesCmdlineFiles(t *testing.T) {
	booter, err := StaticBooter(&types.Spec{
		Kernel:  types.ID("/build/vmlinuz"),
		Initrd:  []types.ID{types.ID("/build/initrd")},
		Cmdline: `kairos.extensions={{ ID "/build/qa-alpha.sysext.raw" }},{{ ID "http://host/d/qa-beta.sysext.raw?t=1" }}`,
	})
	if err != nil {
		t.Fatalf("StaticBooter: %s", err)
	}

	namer, ok := booter.(types.FileNamer)
	if !ok {
		t.Fatalf("the static booter does not implement types.FileNamer")
	}

	for _, tc := range []struct {
		id   types.ID
		want string
	}{
		{"other-0", "qa-alpha.sysext.raw"},
		{"other-1", "qa-beta.sysext.raw"},
		// iPXE loads these with an explicit --name and nothing reads
		// their URL, so they stay unnamed.
		{"kernel", ""},
		{"initrd-0", ""},
		// Nothing the booter was configured with.
		{"other-9", ""},
		{"other-x", ""},
		{"nonsense", ""},
	} {
		if got := namer.BootFileName(tc.id); got != tc.want {
			t.Errorf("BootFileName(%q) = %q, want %q", tc.id, got, tc.want)
		}
	}
}
