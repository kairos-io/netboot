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

package booters

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kairos-io/netboot/types"
	"github.com/kairos-io/netboot/utils"
)

// errorServer serves status on every request, with a non-empty body so the
// response cannot be drained by the transport on its own. It reports every
// connection the server closes on closed.
func errorServer(t *testing.T, status int) (*httptest.Server, <-chan struct{}) {
	t.Helper()

	closed := make(chan struct{}, 8)
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		// A body the caller never reads. Until it is drained or the body is
		// closed, the transport keeps the connection checked out.
		w.Write([]byte(strings.Repeat("the API is unhappy\n", 64))) //nolint:errcheck
	}))
	srv.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateClosed {
			select {
			case closed <- struct{}{}:
			default:
			}
		}
	}
	srv.Start()
	t.Cleanup(srv.Close)

	return srv, closed
}

// signedIDFor mints the kind of ID the apibooter hands out, pointing at target.
func signedIDFor(t *testing.T, b *apibooter, target string) types.ID {
	t.Helper()

	id, err := utils.SignURL(target, &b.key)
	if err != nil {
		t.Fatalf("signing %q: %s", target, err)
	}

	return id
}

func newAPIBooter(t *testing.T, url string) *apibooter {
	t.Helper()

	b, err := APIBooter(url, 5*time.Second)
	if err != nil {
		t.Fatalf("constructing APIBooter: %s", err)
	}

	ab, ok := b.(*apibooter)
	if !ok {
		t.Fatalf("APIBooter returned %T, want *apibooter", b)
	}

	return ab
}

// awaitClose fails unless the server sees the connection go away, which is what
// the client closing the response body causes. A leaked body keeps the
// connection checked out of the pool for as long as the process lives.
func awaitClose(t *testing.T, closed <-chan struct{}, what string) {
	t.Helper()

	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatalf("%s: the response body was never closed, the connection is still checked out", what)
	}
}

func TestAPIBooterReadBootFileClosesBodyOnErrorStatus(t *testing.T) {
	srv, closed := errorServer(t, http.StatusInternalServerError)
	b := newAPIBooter(t, srv.URL+"/")

	id := signedIDFor(t, b, srv.URL+"/kernel")

	body, _, err := b.ReadBootFile(id)
	if err == nil {
		body.Close() //nolint:errcheck
		t.Fatal("ReadBootFile succeeded against a 500, want an error")
	}
	if body != nil {
		t.Fatalf("ReadBootFile returned both an error and a body: %v", err)
	}

	awaitClose(t, closed, "ReadBootFile")
}

func TestAPIBooterWriteBootFileClosesBodyOnErrorStatus(t *testing.T) {
	srv, closed := errorServer(t, http.StatusInternalServerError)
	b := newAPIBooter(t, srv.URL+"/")

	id := signedIDFor(t, b, srv.URL+"/kernel")

	if err := b.WriteBootFile(id, strings.NewReader("payload")); err == nil {
		t.Fatal("WriteBootFile succeeded against a 500, want an error")
	}

	awaitClose(t, closed, "WriteBootFile")
}
