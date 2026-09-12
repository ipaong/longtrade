package paper

import (
	"fmt"
	"math"
	"strings"
)

// ValidationError identifies one invalid user-controlled field.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("paper: invalid %s: %s", e.Field, e.Message)
}

func invalid(field, message string) error {
	return &ValidationError{Field: field, Message: message}
}

// NormalizeSymbol converts beginner-friendly gold symbol spellings to a canonical form.
func NormalizeSymbol(symbol string) string {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	replacer := strings.NewReplacer("/", "", "-", "", "_", "", " ", "")
	return replacer.Replace(symbol)
}

// NormalizeSide removes surrounding whitespace and normalizes letter case.
func NormalizeSide(side Side) Side {
	return Side(strings.ToLower(strings.TrimSpace(string(side))))
}

// ValidateQuote checks the structural guarantees required by the paper engine.
func ValidateQuote(quote Quote) error {
	quote.Symbol = NormalizeSymbol(quote.Symbol)
	if quote.Symbol != SymbolXAUUSD {
		return invalid("symbol", "only XAUUSD is supported in the MVP")
	}
	if !positiveFinite(quote.Bid) {
		return invalid("bid", "must be a positive finite number")
	}
	if !positiveFinite(quote.Ask) {
		return invalid("ask", "must be a positive finite number")
	}
	if quote.Ask < quote.Bid {
		return invalid("ask", "must be greater than or equal to bid")
	}
	if quote.AsOf.IsZero() {
		return invalid("as_of", "is required")
	}
	return nil
}

// OpenPrice returns the executable side of the quote for opening a position.
func (q Quote) OpenPrice(side Side) (float64, error) {
	if err := ValidateQuote(q); err != nil {
		return 0, err
	}
	switch NormalizeSide(side) {
	case SideBuy:
		return q.Ask, nil
	case SideSell:
		return q.Bid, nil
	default:
		return 0, invalid("side", "must be buy or sell")
	}
}

// ClosePrice returns the executable side of the quote for closing a position.
func (q Quote) ClosePrice(side Side) (float64, error) {
	if err := ValidateQuote(q); err != nil {
		return 0, err
	}
	switch NormalizeSide(side) {
	case SideBuy:
		return q.Bid, nil
	case SideSell:
		return q.Ask, nil
	default:
		return 0, invalid("side", "must be buy or sell")
	}
}

// ValidateOpenPositionRequest normalizes and validates a paper market-order request.
func ValidateOpenPositionRequest(request OpenPositionRequest, quote Quote) (OpenPositionRequest, error) {
	request.ClientRequestID = strings.TrimSpace(request.ClientRequestID)
	request.Symbol = NormalizeSymbol(request.Symbol)
	request.Side = NormalizeSide(request.Side)
	quote.Symbol = NormalizeSymbol(quote.Symbol)

	if request.ClientRequestID == "" {
		return request, invalid("client_request_id", "is required")
	}
	if request.Symbol != SymbolXAUUSD {
		return request, invalid("symbol", "only XAUUSD is supported in the MVP")
	}
	if !request.Side.Valid() {
		return request, invalid("side", "must be buy or sell")
	}
	if !positiveFinite(request.VolumeLots) {
		return request, invalid("volume_lots", "must be a positive finite number")
	}
	if err := ValidateQuote(quote); err != nil {
		return request, err
	}
	if request.Symbol != quote.Symbol {
		return request, invalid("symbol", "must match the quote symbol")
	}

	openPrice, err := quote.OpenPrice(request.Side)
	if err != nil {
		return request, err
	}
	if err := ValidateProtection(request.Side, openPrice, request.StopLoss, request.TakeProfit); err != nil {
		return request, err
	}
	return request, nil
}

// ValidateProtection checks stop-loss and take-profit direction against a reference price.
func ValidateProtection(side Side, referencePrice float64, stopLoss, takeProfit *float64) error {
	side = NormalizeSide(side)
	if !side.Valid() {
		return invalid("side", "must be buy or sell")
	}
	if !positiveFinite(referencePrice) {
		return invalid("reference_price", "must be a positive finite number")
	}
	if err := validateOptionalPrice("stop_loss", stopLoss); err != nil {
		return err
	}
	if err := validateOptionalPrice("take_profit", takeProfit); err != nil {
		return err
	}

	switch side {
	case SideBuy:
		if stopLoss != nil && *stopLoss >= referencePrice {
			return invalid("stop_loss", "must be below the reference price for a buy")
		}
		if takeProfit != nil && *takeProfit <= referencePrice {
			return invalid("take_profit", "must be above the reference price for a buy")
		}
	case SideSell:
		if stopLoss != nil && *stopLoss <= referencePrice {
			return invalid("stop_loss", "must be above the reference price for a sell")
		}
		if takeProfit != nil && *takeProfit >= referencePrice {
			return invalid("take_profit", "must be below the reference price for a sell")
		}
	}
	return nil
}

func validateOptionalPrice(field string, value *float64) error {
	if value != nil && !positiveFinite(*value) {
		return invalid(field, "must be a positive finite number")
	}
	return nil
}

func positiveFinite(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}
