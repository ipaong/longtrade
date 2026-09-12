package ta_test

import (
	"testing"

	"github.com/cryptoquantumwave/khunquant/pkg/ta"
)

func TestROC_Basic(t *testing.T) {
	data := []float64{100, 110, 121, 133.1}
	result := ta.ROC(data, 1)
	if result == nil {
		t.Fatal("expected non-nil ROC")
	}
	if len(result) != 3 {
		t.Fatalf("ROC period=1 on 4 values: got len %d, want 3", len(result))
	}
	// (110-100)/100*100 = 10%
	if !almostEqual(result[0], 10.0, 1e-9) {
		t.Errorf("ROC[0] = %.4f, want 10.0", result[0])
	}
}

func TestROC_TooShort(t *testing.T) {
	if ta.ROC([]float64{1, 2}, 5) != nil {
		t.Error("ROC with period >= len should return nil")
	}
}

func TestROC_ZeroPeriod(t *testing.T) {
	if ta.ROC([]float64{1, 2, 3}, 0) != nil {
		t.Error("ROC with period=0 should return nil")
	}
}

func TestROC_ZeroBase(t *testing.T) {
	data := []float64{0, 100, 200}
	result := ta.ROC(data, 1)
	if result == nil {
		t.Fatal("expected non-nil ROC even with zero base")
	}
	// base=0 → result[0] should be 0 (guarded in implementation)
	if result[0] != 0 {
		t.Errorf("ROC with zero base: got %.4f, want 0", result[0])
	}
}

func TestROC_Period2(t *testing.T) {
	data := []float64{100, 110, 120, 130}
	result := ta.ROC(data, 2)
	if result == nil {
		t.Fatal("expected non-nil ROC")
	}
	if len(result) != 2 {
		t.Fatalf("ROC period=2 on 4 values: got len %d, want 2", len(result))
	}
	// (120-100)/100*100 = 20%
	if !almostEqual(result[0], 20.0, 1e-9) {
		t.Errorf("ROC[0] = %.4f, want 20.0", result[0])
	}
}

func TestROC_NegativeChange(t *testing.T) {
	data := []float64{100, 90, 80}
	result := ta.ROC(data, 1)
	if result == nil {
		t.Fatal("expected non-nil ROC")
	}
	// (90-100)/100*100 = -10%
	if !almostEqual(result[0], -10.0, 1e-9) {
		t.Errorf("ROC[0] = %.4f, want -10.0", result[0])
	}
}

func TestMACD_TooShort(t *testing.T) {
	if ta.MACD([]float64{1, 2, 3}, 12, 26, 9) != nil {
		t.Error("MACD with too-short data should return nil")
	}
}

func TestMACD_InvalidParams(t *testing.T) {
	data := make([]float64, 60)
	// fast >= slow is invalid
	if ta.MACD(data, 26, 12, 9) != nil {
		t.Error("MACD with fast >= slow should return nil")
	}
	// zero params invalid
	if ta.MACD(data, 0, 26, 9) != nil {
		t.Error("MACD with fast=0 should return nil")
	}
}

func TestRSI_FlatPrices(t *testing.T) {
	// Constant prices → no gains or losses → RSI should be 100 (avgLoss=0 branch)
	data := make([]float64, 20)
	for i := range data {
		data[i] = 50.0
	}
	result := ta.RSI(data, 14)
	if result == nil {
		t.Fatal("expected non-nil RSI for flat prices")
	}
	// All diffs are 0 → avgLoss=0 → result=100
	if result[0] != 100 {
		t.Errorf("RSI flat prices: got %.2f, want 100", result[0])
	}
}

func TestRSI_Oversold(t *testing.T) {
	// Steadily falling prices → RSI near 0
	data := make([]float64, 30)
	for i := range data {
		data[i] = float64(30 - i)
	}
	result := ta.RSI(data, 14)
	if result == nil {
		t.Fatal("expected non-nil RSI")
	}
	last := result[len(result)-1]
	if last > 5 {
		t.Errorf("expected RSI near 0 for falling data, got %.2f", last)
	}
}
