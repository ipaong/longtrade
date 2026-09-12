package paper

import (
	"errors"
	"math"
	"testing"
	"time"
)

func validQuote() Quote {
	return Quote{
		Symbol: SymbolXAUUSD,
		Bid:    2399.80,
		Ask:    2400.20,
		AsOf:   time.Date(2026, time.September, 12, 10, 0, 0, 0, time.UTC),
	}
}

func floatPointer(value float64) *float64 {
	return &value
}

func validationField(t *testing.T, err error) string {
	t.Helper()
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("error = %v, want *ValidationError", err)
	}
	return validationErr.Field
}

func TestNormalizeSymbol(t *testing.T) {
	for input, want := range map[string]string{
		"XAUUSD":    SymbolXAUUSD,
		"xauusd":    SymbolXAUUSD,
		" XAU/USD ": SymbolXAUUSD,
		"xau-usd":   SymbolXAUUSD,
		"XAU_USD":   SymbolXAUUSD,
	} {
		t.Run(input, func(t *testing.T) {
			if got := NormalizeSymbol(input); got != want {
				t.Fatalf("NormalizeSymbol(%q) = %q, want %q", input, got, want)
			}
		})
	}
}

func TestValidateOpenPositionRequestNormalizesValidBuy(t *testing.T) {
	stopLoss := 2380.0
	takeProfit := 2450.0
	request, err := ValidateOpenPositionRequest(OpenPositionRequest{
		ClientRequestID: " request-1 ",
		Symbol:          " xau/usd ",
		Side:            Side(" BUY "),
		VolumeLots:      0.01,
		StopLoss:        &stopLoss,
		TakeProfit:      &takeProfit,
	}, validQuote())
	if err != nil {
		t.Fatalf("ValidateOpenPositionRequest() error = %v", err)
	}
	if request.ClientRequestID != "request-1" {
		t.Errorf("ClientRequestID = %q, want request-1", request.ClientRequestID)
	}
	if request.Symbol != SymbolXAUUSD {
		t.Errorf("Symbol = %q, want %q", request.Symbol, SymbolXAUUSD)
	}
	if request.Side != SideBuy {
		t.Errorf("Side = %q, want %q", request.Side, SideBuy)
	}
}

func TestValidateOpenPositionRequestRejectsInvalidFields(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(*OpenPositionRequest, *Quote)
		wantField string
	}{
		{
			name: "missing request id",
			mutate: func(request *OpenPositionRequest, _ *Quote) {
				request.ClientRequestID = " "
			},
			wantField: "client_request_id",
		},
		{
			name: "unsupported symbol",
			mutate: func(request *OpenPositionRequest, _ *Quote) {
				request.Symbol = "EURUSD"
			},
			wantField: "symbol",
		},
		{
			name: "invalid side",
			mutate: func(request *OpenPositionRequest, _ *Quote) {
				request.Side = "hold"
			},
			wantField: "side",
		},
		{
			name: "zero lots",
			mutate: func(request *OpenPositionRequest, _ *Quote) {
				request.VolumeLots = 0
			},
			wantField: "volume_lots",
		},
		{
			name: "non-finite lots",
			mutate: func(request *OpenPositionRequest, _ *Quote) {
				request.VolumeLots = math.NaN()
			},
			wantField: "volume_lots",
		},
		{
			name: "quote mismatch",
			mutate: func(_ *OpenPositionRequest, quote *Quote) {
				quote.Symbol = "EURUSD"
			},
			wantField: "symbol",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := OpenPositionRequest{
				ClientRequestID: "request-1",
				Symbol:          SymbolXAUUSD,
				Side:            SideBuy,
				VolumeLots:      0.01,
			}
			quote := validQuote()
			test.mutate(&request, &quote)

			_, err := ValidateOpenPositionRequest(request, quote)
			if err == nil {
				t.Fatal("ValidateOpenPositionRequest() error = nil")
			}
			if got := validationField(t, err); got != test.wantField {
				t.Fatalf("validation field = %q, want %q", got, test.wantField)
			}
		})
	}
}

func TestValidateQuote(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(*Quote)
		wantField string
	}{
		{
			name: "zero bid",
			mutate: func(quote *Quote) {
				quote.Bid = 0
			},
			wantField: "bid",
		},
		{
			name: "ask below bid",
			mutate: func(quote *Quote) {
				quote.Ask = quote.Bid - 0.01
			},
			wantField: "ask",
		},
		{
			name: "missing timestamp",
			mutate: func(quote *Quote) {
				quote.AsOf = time.Time{}
			},
			wantField: "as_of",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			quote := validQuote()
			test.mutate(&quote)
			err := ValidateQuote(quote)
			if err == nil {
				t.Fatal("ValidateQuote() error = nil")
			}
			if got := validationField(t, err); got != test.wantField {
				t.Fatalf("validation field = %q, want %q", got, test.wantField)
			}
		})
	}
}

func TestQuoteUsesBidAskByPositionSide(t *testing.T) {
	quote := validQuote()

	buyOpen, err := quote.OpenPrice(SideBuy)
	if err != nil {
		t.Fatalf("buy OpenPrice() error = %v", err)
	}
	buyClose, err := quote.ClosePrice(SideBuy)
	if err != nil {
		t.Fatalf("buy ClosePrice() error = %v", err)
	}
	sellOpen, err := quote.OpenPrice(SideSell)
	if err != nil {
		t.Fatalf("sell OpenPrice() error = %v", err)
	}
	sellClose, err := quote.ClosePrice(SideSell)
	if err != nil {
		t.Fatalf("sell ClosePrice() error = %v", err)
	}

	if buyOpen != quote.Ask || buyClose != quote.Bid {
		t.Fatalf("buy prices = open %.2f close %.2f, want ask %.2f bid %.2f", buyOpen, buyClose, quote.Ask, quote.Bid)
	}
	if sellOpen != quote.Bid || sellClose != quote.Ask {
		t.Fatalf("sell prices = open %.2f close %.2f, want bid %.2f ask %.2f", sellOpen, sellClose, quote.Bid, quote.Ask)
	}
}

func TestValidateProtectionDirection(t *testing.T) {
	tests := []struct {
		name       string
		side       Side
		stopLoss   *float64
		takeProfit *float64
		wantField  string
	}{
		{
			name:       "valid buy",
			side:       SideBuy,
			stopLoss:   floatPointer(2390),
			takeProfit: floatPointer(2420),
		},
		{
			name:      "buy stop above price",
			side:      SideBuy,
			stopLoss:  floatPointer(2410),
			wantField: "stop_loss",
		},
		{
			name:       "buy target below price",
			side:       SideBuy,
			takeProfit: floatPointer(2390),
			wantField:  "take_profit",
		},
		{
			name:       "valid sell",
			side:       SideSell,
			stopLoss:   floatPointer(2410),
			takeProfit: floatPointer(2390),
		},
		{
			name:      "sell stop below price",
			side:      SideSell,
			stopLoss:  floatPointer(2390),
			wantField: "stop_loss",
		},
		{
			name:       "sell target above price",
			side:       SideSell,
			takeProfit: floatPointer(2410),
			wantField:  "take_profit",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateProtection(test.side, 2400, test.stopLoss, test.takeProfit)
			if test.wantField == "" {
				if err != nil {
					t.Fatalf("ValidateProtection() error = %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("ValidateProtection() error = nil")
			}
			if got := validationField(t, err); got != test.wantField {
				t.Fatalf("validation field = %q, want %q", got, test.wantField)
			}
		})
	}
}
