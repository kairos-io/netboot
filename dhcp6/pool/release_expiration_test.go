package pool

import (
	"math/big"
	"net"
	"testing"
	"time"
)

// reserveOne reserves a single association and returns it, failing the test if
// the pool could not hand one out.
func reserveOne(t *testing.T, p *RandomAddressPool, clientID, iaID []byte) net.IP {
	t.Helper()

	ias, err := p.ReserveAddresses(clientID, [][]byte{iaID})
	if err != nil {
		t.Fatalf("reserving: %s", err)
	}
	if len(ias) != 1 {
		t.Fatalf("reserving returned %d associations, want 1", len(ias))
	}

	return net.IP(ias[0].IPAddress)
}

// A released association keeps its place in the expiration queue, and the queue
// entry is keyed by the client and interface id, which is exactly the key the
// client gets again when it comes back. The stale timer must not revoke the
// lease that holds the key now.
func TestStaleExpirationDoesNotRevokeTheCurrentLease(t *testing.T) {
	clientID := []byte("Client-id")
	iaID := []byte("interface-id")

	now := time.Now()
	pool := NewRandomAddressPool(net.ParseIP("2001:db8:f00f:cafe::1"), 16, 100)
	pool.timeNow = func() time.Time { return now }

	reserveOne(t, pool, clientID, iaID)
	pool.ReleaseAddresses(clientID, [][]byte{iaID})

	// The client comes back 10s later, so its new lease runs to t=110 while
	// the released one's queue entry still says t=100.
	now = now.Add(10 * time.Second)
	current := reserveOne(t, pool, clientID, iaID)

	// Past the released lease's expiry, 5s short of the current one's. Any
	// pool activity runs the sweep.
	now = now.Add(95 * time.Second)
	renewed := reserveOne(t, pool, clientID, iaID)

	if !renewed.Equal(current) {
		t.Fatalf("the client's still-valid lease on %s was revoked early and replaced with %s", current, renewed)
	}
}

// The other half of the same mistake: the stale entry carries the address the
// release already freed, so acting on it hands that address out while the
// current one stays marked used with nothing behind it.
func TestStaleExpirationLeavesThePoolBookkeepingIntact(t *testing.T) {
	clientID := []byte("Client-id")
	iaID := []byte("interface-id")

	now := time.Now()
	pool := NewRandomAddressPool(net.ParseIP("2001:db8:f00f:cafe::1"), 16, 100)
	pool.timeNow = func() time.Time { return now }

	reserveOne(t, pool, clientID, iaID)
	pool.ReleaseAddresses(clientID, [][]byte{iaID})

	now = now.Add(10 * time.Second)
	current := reserveOne(t, pool, clientID, iaID)

	now = now.Add(95 * time.Second)
	pool.ReserveAddresses([]byte("someone-else"), [][]byte{[]byte("other-interface")}) //nolint:errcheck

	hash := pool.calculateIAIDHash(clientID, iaID)
	if _, exists := pool.identityAssociations[hash]; !exists {
		t.Fatal("the client's association was dropped by the released lease's expiration")
	}
	if _, used := pool.usedIps[big.NewInt(0).SetBytes(current).Uint64()]; !used {
		t.Fatalf("%s is still leased but no longer marked used, so the pool can hand it to a second client", current)
	}
}

// The guard must not stop a lease nobody released from expiring on time.
func TestAnUnreleasedAssociationStillExpires(t *testing.T) {
	clientID := []byte("Client-id")
	iaID := []byte("interface-id")

	now := time.Now()
	pool := NewRandomAddressPool(net.ParseIP("2001:db8:f00f:cafe::1"), 16, 100)
	pool.timeNow = func() time.Time { return now }

	first := reserveOne(t, pool, clientID, iaID)

	now = now.Add(101 * time.Second)
	pool.ReserveAddresses([]byte("someone-else"), [][]byte{[]byte("other-interface")}) //nolint:errcheck

	hash := pool.calculateIAIDHash(clientID, iaID)
	if _, exists := pool.identityAssociations[hash]; exists {
		t.Fatal("an expired association was kept")
	}
	if _, used := pool.usedIps[big.NewInt(0).SetBytes(first).Uint64()]; used {
		t.Fatalf("%s expired but was never returned to the pool", first)
	}
}
