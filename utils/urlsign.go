// Copyright 2016 Google Inc.
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

package utils

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"github.com/kairos-io/netboot/types"
	"golang.org/x/crypto/nacl/secretbox"
)

// SignURL constructs an ID from u, signed with key.
func SignURL(u string, key *[32]byte) (types.ID, error) {
	var nonce [24]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return "", fmt.Errorf("could not read randomness for signing nonce: %s", err)
	}

	out := nonce[:]

	// Secretbox is authenticated encryption. In theory we only need
	// symmetric authentication, but secretbox is stupidly simple to
	// use and hard to get wrong, and the encryption overhead should
	// be tiny for such a small URL unless you're trying to
	// simultaneously netboot a million machines. This is one case
	// where convenience and certainty that you got it right trumps
	// pure efficiency.
	out = secretbox.Seal(out, []byte(u), &nonce, key)
	return types.ID(base64.URLEncoding.EncodeToString(out)), nil
}

// GetURL returns the URL contained within id.
//
// id must have been created by signURL, with key.
func GetURL(id types.ID, key *[32]byte) (string, error) {
	signed, err := base64.URLEncoding.DecodeString(string(id))
	if err != nil {
		return "", err
	}
	if len(signed) < 24 {
		return "", errors.New("signed blob too short to be valid")
	}

	var nonce [24]byte
	copy(nonce[:], signed)
	out, ok := secretbox.Open(nil, signed[24:], &nonce, key)
	if !ok {
		return "", errors.New("signature verification failed")
	}
	return string(out), nil
}
