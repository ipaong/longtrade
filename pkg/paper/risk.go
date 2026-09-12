package paper

import (
	"fmt"
	"math"
)

// RiskLimits are deterministic guardrails for the local paper account.
type RiskLimits struct {
	MaxOpenPositions int
	MaxVolumeLots    float64
	MaxDailyLossUSD  float64
}

func DefaultRiskLimits() RiskLimits {
	return RiskLimits{MaxOpenPositions: 10, MaxVolumeLots: 1, MaxDailyLossUSD: 1_000}
}

func validateRiskLimits(limits RiskLimits) error {
	if limits.MaxOpenPositions <= 0 || !positiveFinite(limits.MaxVolumeLots) || !positiveFinite(limits.MaxDailyLossUSD) {
		return fmt.Errorf("paper: invalid risk limits")
	}
	return nil
}

func validateLot(instrument Instrument, lots float64) error {
	if !positiveFinite(lots) || lots < instrument.MinimumLot || lots > instrument.MaximumLot {
		return invalid("volume_lots", fmt.Sprintf("must be between %.2f and %.2f", instrument.MinimumLot, instrument.MaximumLot))
	}
	steps := (lots - instrument.MinimumLot) / instrument.LotStep
	if math.Abs(steps-math.Round(steps)) > 1e-8 {
		return invalid("volume_lots", fmt.Sprintf("must use %.2f lot steps", instrument.LotStep))
	}
	return nil
}
