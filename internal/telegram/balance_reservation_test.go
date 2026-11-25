package telegram

import (
	"errors"
	"sync"
	"testing"
)

func TestReserveSubtractsInFlightPayments(t *testing.T) {
	state := &senderReservations{}
	balance := func() (int64, error) { return 1000, nil }

	first, available, err := state.reserve(600, balance)
	if err != nil || first == nil || available != 1000 {
		t.Fatalf("first reserve = %v, %d, %v; want a reservation with 1000 available", first, available, err)
	}

	// The first payment is still in flight, so only 400 is left.
	second, available, err := state.reserve(600, balance)
	if err != nil || second != nil || available != 400 {
		t.Fatalf("second reserve = %v, %d, %v; want no reservation with 400 available", second, available, err)
	}

	// Releasing (twice, as early release plus defer does) frees it exactly once.
	first.Release()
	first.Release()
	if state.reserved != 0 {
		t.Fatalf("reserved = %d after release, want 0", state.reserved)
	}
	if third, _, _ := state.reserve(600, balance); third == nil {
		t.Fatal("reserve after release should succeed")
	}
}

func TestReserveBalanceError(t *testing.T) {
	state := &senderReservations{}
	wantErr := errors.New("lnbits down")
	res, _, err := state.reserve(1, func() (int64, error) { return 0, wantErr })
	if res != nil || !errors.Is(err, wantErr) {
		t.Fatalf("reserve = %v, %v; want nil reservation and the lookup error", res, err)
	}
	if state.reserved != 0 {
		t.Fatalf("reserved = %d after failed lookup, want 0", state.reserved)
	}
}

// Many concurrent payments must never reserve more than the balance.
func TestReserveConcurrent(t *testing.T) {
	state := &senderReservations{}
	balance := func() (int64, error) { return 1000, nil }

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		granted int
	)
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if res, _, _ := state.reserve(100, balance); res != nil {
				mu.Lock()
				granted++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if granted != 10 {
		t.Fatalf("granted %d reservations of 100 from a balance of 1000, want 10", granted)
	}
}

func TestReservedBalance(t *testing.T) {
	const id int64 = 987654321
	// reservationsFor below adds a global entry; remove it so -count=N and
	// later tests start from no state for this user.
	t.Cleanup(func() {
		reservationsMu.Lock()
		delete(reservations, id)
		reservationsMu.Unlock()
	})
	if got := ReservedBalance(id); got != 0 {
		t.Fatalf("unknown user: %d, want 0", got)
	}
	reservationsMu.Lock()
	_, created := reservations[id]
	reservationsMu.Unlock()
	if created {
		t.Fatal("ReservedBalance created state for an unknown user")
	}

	r, _, err := reservationsFor(id).reserve(300, func() (int64, error) { return 1000, nil })
	if err != nil || r == nil {
		t.Fatalf("reserve: %v %v", r, err)
	}
	if got := ReservedBalance(id); got != 300 {
		t.Fatalf("in flight: %d, want 300", got)
	}
	r.Release()
	if got := ReservedBalance(id); got != 0 {
		t.Fatalf("after release: %d, want 0", got)
	}
}
