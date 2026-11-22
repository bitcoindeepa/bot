package price

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

type trackingBody struct {
	reader io.Reader
	closed bool
}

func (b *trackingBody) Read(p []byte) (int, error) { return b.reader.Read(p) }
func (b *trackingBody) Close() error               { b.closed = true; return nil }

func newTestWatcher(status int, payload string) (*PriceWatcher, *trackingBody) {
	body := &trackingBody{reader: strings.NewReader(payload)}
	pw := &PriceWatcher{client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: status, Body: body, Header: make(http.Header)}, nil
	})}}
	return pw, body
}

func TestGetCoinbasePriceReadsAndClosesBody(t *testing.T) {
	pw, body := newTestWatcher(http.StatusOK, `{"data":{"amount":"1234.56"}}`)
	price, err := pw.GetCoinbasePrice("USD")
	if err != nil {
		t.Fatalf("GetCoinbasePrice: %v", err)
	}
	if price != 1234.56 {
		t.Errorf("price = %v, want 1234.56", price)
	}
	if !body.closed {
		t.Error("response body was not closed")
	}
}

func TestGetCoinbasePriceRejectsNonOKStatus(t *testing.T) {
	pw, body := newTestWatcher(http.StatusServiceUnavailable, `{"data":{"amount":"1"}}`)
	if _, err := pw.GetCoinbasePrice("USD"); err == nil {
		t.Fatal("GetCoinbasePrice succeeded on a 503 response")
	}
	if !body.closed {
		t.Error("response body was not closed")
	}
}

func TestGetCoinbasePriceMissingAmount(t *testing.T) {
	pw, _ := newTestWatcher(http.StatusOK, `{"data":{}}`)
	if _, err := pw.GetCoinbasePrice("USD"); err == nil {
		t.Fatal("GetCoinbasePrice succeeded without an amount")
	}
}

func TestGetBitfinexPriceReadsAndClosesBody(t *testing.T) {
	pw, body := newTestWatcher(http.StatusOK, `{"last_price":"42.5"}`)
	price, err := pw.GetBitfinexPrice("USD")
	if err != nil {
		t.Fatalf("GetBitfinexPrice: %v", err)
	}
	if price != 42.5 {
		t.Errorf("price = %v, want 42.5", price)
	}
	if !body.closed {
		t.Error("response body was not closed")
	}
}

func TestGetBitfinexPriceRejectsNonOKStatus(t *testing.T) {
	pw, body := newTestWatcher(http.StatusInternalServerError, `{"last_price":"1"}`)
	if _, err := pw.GetBitfinexPrice("USD"); err == nil {
		t.Fatal("GetBitfinexPrice succeeded on a 500 response")
	}
	if !body.closed {
		t.Error("response body was not closed")
	}
}
