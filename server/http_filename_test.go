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

package server

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kairos-io/netboot/booters"
	"github.com/kairos-io/netboot/types"
)

// namingBooter is a Booter that knows a name for every ID it is given, which
// is what types.FileNamer is for.
type namingBooter struct {
	spec  *types.Spec
	names map[types.ID]string
}

func (b namingBooter) BootSpec(types.Machine) (*types.Spec, error) { return b.spec, nil }
func (b namingBooter) ReadBootFile(id types.ID) (io.ReadCloser, int64, error) {
	d := fmt.Sprintf("contents of %s", id)
	return io.NopCloser(bytes.NewBufferString(d)), int64(len(d)), nil
}
func (b namingBooter) WriteBootFile(types.ID, io.Reader) error { return errors.New("no") }
func (b namingBooter) BootFileName(id types.ID) string         { return b.names[id] }

func ipxeFor(t *testing.T, booter types.Booter) string {
	t.Helper()
	log := func(subsystem, msg string) { t.Logf("[%s] %s", subsystem, msg) }
	s := &Server{Booter: booter, Log: log, Debug: log, events: make(map[string][]machineEvent)}

	rr := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/_/ipxe?mac=01:02:03:04:05:06&arch=0", nil)
	if err != nil {
		t.Fatalf("Constructing ipxe request: %s", err)
	}
	req.Host = "localhost:1234"
	s.handleIpxe(rr, req)
	if rr.Code != 200 {
		t.Fatalf("Got HTTP %d from the ipxe request, expected 200", rr.Code)
	}
	return rr.Body.String()
}

// The regression behind kairos-io/kairos#5369: two files handed to the booted
// system on the cmdline were served from the same path, /_/file, with only the
// query telling them apart. A client that names a download after the URL path,
// which is all a kernel cmdline carries, called both of them "file", so the
// second collided with the first and neither was usable. Each now gets a path
// that ends in its own name.
func TestIpxeCmdlineURLsCarryTheFileName(t *testing.T) {
	booter := namingBooter{
		spec: &types.Spec{
			Kernel:  types.ID("kernel"),
			Initrd:  []types.ID{types.ID("initrd-0")},
			Cmdline: `kairos.extensions={{ ID "other-0" }},{{ ID "other-1" }}`,
		},
		names: map[types.ID]string{
			"other-0": "qa-alpha.sysext.raw",
			"other-1": "qa-beta.sysext.raw",
		},
	}

	script := ipxeFor(t, booter)

	for _, want := range []string{
		"http://localhost:1234/_/file/qa-alpha.sysext.raw?name=other-0",
		"http://localhost:1234/_/file/qa-beta.sysext.raw?name=other-1",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("iPXE script does not carry %q:\n%s", want, script)
		}
	}
}

// A name the booter does not know keeps the URL it always had, so a Booter
// that does not implement types.FileNamer, and an ID that booter has no name
// for, both keep working.
func TestIpxeCmdlineURLUnnamedFileKeepsThePlainPath(t *testing.T) {
	booter := namingBooter{
		spec: &types.Spec{
			Kernel:  types.ID("kernel"),
			Cmdline: `thing={{ ID "other-0" }}`,
		},
		names: map[types.ID]string{},
	}

	script := ipxeFor(t, booter)

	want := "thing=http://localhost:1234/_/file?name=other-0"
	if !strings.Contains(script, want) {
		t.Errorf("iPXE script does not carry %q:\n%s", want, script)
	}
}

// A name is escaped as one path segment, so it cannot smuggle a slash or a
// question mark into the URL and change which file is asked for.
func TestFileURLEscapesTheName(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   string
		file string
		want string
	}{
		{"plain", "other-0", "a.raw", "http://h/_/file/a.raw?name=other-0"},
		{"space", "other-0", "a b.raw", "http://h/_/file/a%20b.raw?name=other-0"},
		{"slash", "other-0", "a/b.raw", "http://h/_/file/a%2Fb.raw?name=other-0"},
		{"query", "other-0", "a?b.raw", "http://h/_/file/a%3Fb.raw?name=other-0"},
		{"escaped id", "other 0", "a.raw", "http://h/_/file/a.raw?name=other+0"},
		{"no name", "other-0", "", "http://h/_/file?name=other-0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := fileURL("h", tc.id, tc.file); got != tc.want {
				t.Errorf("fileURL(%q, %q) = %q, want %q", tc.id, tc.file, got, tc.want)
			}
		})
	}
}

// The named URL has to reach the same handler as the plain one, and resolve by
// the query, not by the path segment: the name is for the client's benefit
// only.
func TestNamedFilePathIsServed(t *testing.T) {
	log := func(subsystem, msg string) { t.Logf("[%s] %s", subsystem, msg) }
	s := &Server{Booter: readBootFile("stuff"), Log: log, Debug: log}
	mux := http.NewServeMux()
	s.serveHTTP(mux)

	for _, tc := range []struct {
		path string
		want string
	}{
		{"/_/file?name=test", "test stuff"},
		{"/_/file/qa-alpha.sysext.raw?name=test", "test stuff"},
		{"/_/file/a%20b.raw?name=quux", "quux stuff"},
	} {
		rr := httptest.NewRecorder()
		req, err := http.NewRequest("GET", tc.path, nil)
		if err != nil {
			t.Fatalf("Constructing request for %s: %s", tc.path, err)
		}
		mux.ServeHTTP(rr, req)

		if rr.Code != 200 {
			t.Fatalf("GET %s: got HTTP %d, expected 200", tc.path, rr.Code)
		}
		if rr.Body.String() != tc.want {
			t.Errorf("GET %s: got %q, want %q", tc.path, rr.Body.String(), tc.want)
		}
	}
}

// A named path with no name query is still a bad request: the path segment is
// decoration, never the lookup key.
func TestNamedFilePathStillNeedsTheQuery(t *testing.T) {
	log := func(subsystem, msg string) { t.Logf("[%s] %s", subsystem, msg) }
	s := &Server{Booter: readBootFile("stuff"), Log: log, Debug: log}
	mux := http.NewServeMux()
	s.serveHTTP(mux)

	rr := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/_/file/qa-alpha.sysext.raw", nil)
	if err != nil {
		t.Fatalf("Constructing request: %s", err)
	}
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("Got HTTP %d, expected %d", rr.Code, http.StatusBadRequest)
	}
}

// The whole path, with the booter the netboot server actually runs: two local
// images declared on the cmdline are advertised under their own names, and
// each named URL serves the image it names. This is the netboot half of
// kairos-io/kairos#5040, exercised end to end.
func TestStaticBooterServesCmdlineFilesByName(t *testing.T) {
	dir := t.TempDir()
	images := map[string]string{
		"qa-alpha.sysext.raw": "alpha contents",
		"qa-beta.sysext.raw":  "beta contents",
	}
	for name, contents := range images {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(contents), 0644); err != nil {
			t.Fatalf("writing %s: %s", name, err)
		}
	}
	kernel := filepath.Join(dir, "vmlinuz")
	if err := os.WriteFile(kernel, []byte("kernel contents"), 0644); err != nil {
		t.Fatalf("writing the kernel: %s", err)
	}

	booter, err := booters.StaticBooter(&types.Spec{
		Kernel: types.ID(kernel),
		Cmdline: fmt.Sprintf(`kairos.extensions={{ ID %q }},{{ ID %q }}`,
			filepath.Join(dir, "qa-alpha.sysext.raw"),
			filepath.Join(dir, "qa-beta.sysext.raw")),
	})
	if err != nil {
		t.Fatalf("StaticBooter: %s", err)
	}

	log := func(subsystem, msg string) { t.Logf("[%s] %s", subsystem, msg) }
	s := &Server{Booter: booter, Log: log, Debug: log, events: make(map[string][]machineEvent)}
	mux := http.NewServeMux()
	s.serveHTTP(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "http://")
	rr := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/_/ipxe?mac=01:02:03:04:05:06&arch=0", nil)
	if err != nil {
		t.Fatalf("Constructing ipxe request: %s", err)
	}
	req.Host = host
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("Got HTTP %d from the ipxe request, expected 200", rr.Code)
	}
	script := rr.Body.String()

	for name, contents := range images {
		want := fmt.Sprintf("http://%s/_/file/%s?name=other-", host, name)
		if !strings.Contains(script, want) {
			t.Fatalf("iPXE script does not advertise %s under its own name:\n%s", name, script)
		}

		// The URL the booted system would see for this image: take it
		// out of the script and fetch it, the way the agent does.
		at := strings.Index(script, want)
		url := script[at:]
		// The cmdline separates the two extensions with a comma, and
		// the kernel cmdline separates arguments with a space.
		if end := strings.IndexAny(url, ", \n"); end >= 0 {
			url = url[:end]
		}
		resp, err := http.Get(url)
		if err != nil {
			t.Fatalf("GET %s: %s", url, err)
		}
		body, err := io.ReadAll(resp.Body)
		if closeErr := resp.Body.Close(); closeErr != nil {
			t.Fatalf("closing %s: %s", url, closeErr)
		}
		if err != nil {
			t.Fatalf("reading %s: %s", url, err)
		}
		if resp.StatusCode != 200 {
			t.Fatalf("GET %s: HTTP %d", url, resp.StatusCode)
		}
		if string(body) != contents {
			t.Errorf("GET %s served %q, want %q", url, body, contents)
		}
	}
}

// A Booter with no name lookup at all gets one that names nothing, rather than
// a nil function the caller would have to guard.
func TestBootFileNamerWithoutTheCapability(t *testing.T) {
	namer := bootFileNamer(readBootFile("stuff"))
	if got := namer(types.ID("other-0")); got != "" {
		t.Errorf("bootFileNamer on a plain Booter named %q, want the empty string", got)
	}
}
