package api

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/LightningTipBot/LightningTipBot/internal/lnbits"
)

func TestBalanceBreakdown(t *testing.T) {
	pots := []lnbits.SavingsPot{
		{Name: "rent", Balance: 3000},
		{Name: "trip", Balance: 1500},
	}
	available, potTotal, list := balanceBreakdown(10000, pots)
	if available != 5500 || potTotal != 4500 {
		t.Fatalf("available=%d pots=%d, want 5500/4500", available, potTotal)
	}
	if len(list) != 2 || list[0] != (PotBalance{Name: "rent", Balance: 3000}) || list[1] != (PotBalance{Name: "trip", Balance: 1500}) {
		t.Fatalf("pot list %+v", list)
	}
}

func TestBalanceBreakdownNoPots(t *testing.T) {
	available, potTotal, list := balanceBreakdown(7000, nil)
	if available != 7000 || potTotal != 0 {
		t.Fatalf("available=%d pots=%d, want 7000/0", available, potTotal)
	}
	if list == nil || len(list) != 0 {
		t.Fatalf("want empty non-nil list, got %#v", list)
	}
}

// Pots larger than the wallet (e.g. wallet drained by an older code path)
// must not report negative spendable sats.
func TestBalanceBreakdownNeverNegative(t *testing.T) {
	available, potTotal, _ := balanceBreakdown(1000, []lnbits.SavingsPot{{Name: "x", Balance: 4000}})
	if available != 0 || potTotal != 4000 {
		t.Fatalf("available=%d pots=%d, want 0/4000", available, potTotal)
	}
}

// The backend relies on these keys being present even when they are zero.
func TestUserBalanceResponseAlwaysHasPotFields(t *testing.T) {
	_, _, list := balanceBreakdown(0, nil)
	b, err := json.Marshal(UserBalanceResponse{Success: true, TelegramID: 1, Pots: list})
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{`"balance":0`, `"available_balance":0`, `"pot_balance":0`, `"pots":[]`} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %s in %s", want, s)
		}
	}
}
