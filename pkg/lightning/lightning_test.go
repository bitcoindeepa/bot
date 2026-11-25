package lightning

import (
	"strings"
	"testing"
)

// data stands in for an invoice's data part. IsInvoice only checks the shape,
// so the bytes don't need to form a valid checksum or signature.
var data = strings.Repeat("qpzry9x8gf2tvdw0s3jn54khce6mua7l", 8)

func TestIsInvoice(t *testing.T) {
	mainnet := "lnbc10u1" + data

	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"mainnet with amount", mainnet, true},
		{"mainnet without amount", "lnbc1" + data, true},
		{"uppercase, as QR codes encode it", strings.ToUpper(mainnet), true},
		{"lightning: URI", "lightning:" + mainnet, true},
		{"LIGHTNING: URI uppercase", "LIGHTNING:" + strings.ToUpper(mainnet), true},
		{"testnet", "lntb2500n1" + data, true},
		{"signet", "lntbs1m1" + data, true},
		{"regtest", "lnbcrt50u1" + data, true},
		{"simnet", "lnsb1p1" + data, true},
		// Malformed invoices must still be routed to the decoder, which
		// reports a precise error (see internal/api TestSendRejectsBadRequests).
		{"undecodable but invoice-shaped", "lnbc1notarealinvoice", true},

		{"empty", "", false},
		{"bare prefix", "lnbc", false},
		{"no separator", "lnbcfoo", false},
		{"prefix only with URI", "lightning:", false},
		{"trailing text", mainnet + " thanks", false},
		{"leading space", " " + mainnet, false},
		{"newline inside", "lnbc1" + data + "\nlnbc1" + data, false},
		{"punctuation in data", "lnbc1abc.def", false},
		{"unknown network", "lnxx10u1" + data, false},
		{"lnurl", "lnurl1dp68gurn8ghj7um9wfmxjcm99e3k7mf0v9cxj0m385ekvcenxc6r2c35xvukxefcv5mkvv34x5ekzd3ev56nyd3hxqurzepexejxxepnxscrvwfnv9nxzcn9xq6xyefhvgcxxcmyxymnserxfq5fns", false},
		{"on-chain bech32 address", "bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t4", false},
		{"telegram username", "@lnbc1user", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsInvoice(tt.in); got != tt.want {
				t.Errorf("IsInvoice(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestIsLightningAddress(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"alice@example.com", true},
		{"Alice@Example.COM", true},
		{"tips+bot@pay.sub-domain.example.lk", true},
		{"first.last_1@wallet.example", true},

		{"", false},
		{"alice", false},
		{"@alice", false},
		{"alice@", false},
		{"alice@localhost", false},
		{"Alice <alice@example.com>", false},
		{"<alice@example.com>", false},
		{`"a b"@example.com`, false},
		{" alice@example.com", false},
		{"alice@example.com ", false},
		{"alice@bob@example.com", false},
		{"alice@example..com", false},
		{"alice@.example.com", false},
		{"alice@example.com/path", false},
		{"alice@example.com?x=1", false},
	}
	for _, tt := range tests {
		if got := IsLightningAddress(tt.in); got != tt.want {
			t.Errorf("IsLightningAddress(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
