package dca

import (
	"testing"
)

// TestFloatParam tests the floatParam function
func TestFloatParam(t *testing.T) {
	tests := []struct {
		name string
		params map[string]any
		key string
		def float64
		want float64
	}{
		{
			name: "float64 value",
			params: map[string]any{"threshold": float64(0.5)},
			key: "threshold",
			def: 0.0,
			want: 0.5,
		},
		{
			name: "missing key",
			params: map[string]any{"other": float64(0.5)},
			key: "threshold",
			def: 1.0,
			want: 1.0,
		},
		{
			name: "non-float64 value",
			params: map[string]any{"threshold": "not-a-float"},
			key: "threshold",
			def: 2.0,
			want: 2.0,
		},
		{
			name: "int value (not converted)",
			params: map[string]any{"threshold": 42},
			key: "threshold",
			def: 1.0,
			want: 1.0,
		},
		{
			name: "empty params",
			params: map[string]any{},
			key: "threshold",
			def: 3.0,
			want: 3.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := floatParam(tt.params, tt.key, tt.def)
			if got != tt.want {
				t.Errorf("floatParam(%v, %q, %v) = %v, want %v", tt.params, tt.key, tt.def, got, tt.want)
			}
		})
	}
}

// TestIntParam tests the intParam function
func TestIntParam(t *testing.T) {
	tests := []struct {
		name string
		params map[string]any
		key string
		def int
		want int
	}{
		{
			name: "float64 value",
			params: map[string]any{"period": float64(14)},
			key: "period",
			def: 0,
			want: 14,
		},
		{
			name: "int value",
			params: map[string]any{"period": 21},
			key: "period",
			def: 0,
			want: 21,
		},
		{
			name: "int64 value",
			params: map[string]any{"period": int64(28)},
			key: "period",
			def: 0,
			want: 28,
		},
		{
			name: "missing key",
			params: map[string]any{"other": 42},
			key: "period",
			def: 14,
			want: 14,
		},
		{
			name: "string value",
			params: map[string]any{"period": "14"},
			key: "period",
			def: 7,
			want: 7,
		},
		{
			name: "empty params",
			params: map[string]any{},
			key: "period",
			def: 14,
			want: 14,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := intParam(tt.params, tt.key, tt.def)
			if got != tt.want {
				t.Errorf("intParam(%v, %q, %d) = %d, want %d", tt.params, tt.key, tt.def, got, tt.want)
			}
		})
	}
}

// TestComputeIndicatorValues tests the ComputeIndicatorValues function
func TestComputeIndicatorValues_EmptyData(t *testing.T) {
	spec := IndicatorSpec{
		Alias: "test_rsi",
		Kind: "rsi",
		Params: map[string]any{"period": 14},
	}

	_, _, err := ComputeIndicatorValues(spec, []float64{}, []float64{}, []float64{}, []float64{}, []float64{})
	if err == nil {
		t.Fatal("expected error for empty data, got nil")
	}
}

// TestComputeIndicatorValues_RSI tests RSI indicator computation
func TestComputeIndicatorValues_RSI(t *testing.T) {
	spec := IndicatorSpec{
		Alias: "test_rsi",
		Kind: "rsi",
		Params: map[string]any{"period": 2},
	}

	closes := []float64{100, 102, 101, 103, 102, 104}
	opens := make([]float64, len(closes))
	highs := make([]float64, len(closes))
	lows := make([]float64, len(closes))
	volumes := make([]float64, len(closes))

	curr, prev, err := ComputeIndicatorValues(spec, closes, opens, highs, lows, volumes)
	if err != nil {
		t.Fatalf("ComputeIndicatorValues() error: %v", err)
	}

	// Should return float64 values for RSI
	if _, ok := curr.(float64); !ok {
		t.Errorf("expected float64, got %T", curr)
	}
	if _, ok := prev.(float64); !ok {
		t.Errorf("expected float64, got %T", prev)
	}
}

// TestComputeIndicatorValues_Unsupported tests unsupported indicator
func TestComputeIndicatorValues_Unsupported(t *testing.T) {
	spec := IndicatorSpec{
		Alias: "test_unknown",
		Kind: "unknown",
		Params: map[string]any{},
	}

	closes := []float64{100, 102, 101}
	opens := make([]float64, len(closes))
	highs := make([]float64, len(closes))
	lows := make([]float64, len(closes))
	volumes := make([]float64, len(closes))

	_, _, err := ComputeIndicatorValues(spec, closes, opens, highs, lows, volumes)
	if err == nil {
		t.Fatal("expected error for unsupported indicator, got nil")
	}
}

func makeOHLCV(n int) (closes, opens, highs, lows, volumes []float64) {
	closes = make([]float64, n)
	opens = make([]float64, n)
	highs = make([]float64, n)
	lows = make([]float64, n)
	volumes = make([]float64, n)
	for i := range n {
		closes[i] = 100 + float64(i)
		opens[i] = 99 + float64(i)
		highs[i] = 101 + float64(i)
		lows[i] = 98 + float64(i)
		volumes[i] = 1000
	}
	return
}

func TestComputeIndicatorValues_SMA(t *testing.T) {
	closes, opens, highs, lows, volumes := makeOHLCV(30)
	spec := IndicatorSpec{Alias: "my_sma", Kind: "sma", Params: map[string]any{"period": float64(5)}}
	curr, prev, err := ComputeIndicatorValues(spec, closes, opens, highs, lows, volumes)
	if err != nil {
		t.Fatalf("sma error: %v", err)
	}
	if _, ok := curr.(float64); !ok {
		t.Errorf("sma curr type = %T, want float64", curr)
	}
	if _, ok := prev.(float64); !ok {
		t.Errorf("sma prev type = %T, want float64", prev)
	}
}

func TestComputeIndicatorValues_EMA(t *testing.T) {
	closes, opens, highs, lows, volumes := makeOHLCV(30)
	spec := IndicatorSpec{Alias: "my_ema", Kind: "ema", Params: map[string]any{"period": float64(5)}}
	_, _, err := ComputeIndicatorValues(spec, closes, opens, highs, lows, volumes)
	if err != nil {
		t.Fatalf("ema error: %v", err)
	}
}

func TestComputeIndicatorValues_MACD(t *testing.T) {
	closes, opens, highs, lows, volumes := makeOHLCV(60)
	spec := IndicatorSpec{Alias: "my_macd", Kind: "macd", Params: map[string]any{}}
	curr, _, err := ComputeIndicatorValues(spec, closes, opens, highs, lows, volumes)
	if err != nil {
		t.Fatalf("macd error: %v", err)
	}
	if _, ok := curr.(map[string]float64); !ok {
		t.Errorf("macd curr type = %T, want map[string]float64", curr)
	}
}

func TestComputeIndicatorValues_BB(t *testing.T) {
	closes, opens, highs, lows, volumes := makeOHLCV(30)
	spec := IndicatorSpec{Alias: "my_bb", Kind: "bb", Params: map[string]any{}}
	curr, _, err := ComputeIndicatorValues(spec, closes, opens, highs, lows, volumes)
	if err != nil {
		t.Fatalf("bb error: %v", err)
	}
	if _, ok := curr.(map[string]float64); !ok {
		t.Errorf("bb curr type = %T, want map[string]float64", curr)
	}
}

func TestComputeIndicatorValues_ATR(t *testing.T) {
	closes, opens, highs, lows, volumes := makeOHLCV(30)
	spec := IndicatorSpec{Alias: "my_atr", Kind: "atr", Params: map[string]any{"period": float64(14)}}
	_, _, err := ComputeIndicatorValues(spec, closes, opens, highs, lows, volumes)
	if err != nil {
		t.Fatalf("atr error: %v", err)
	}
}

func TestComputeIndicatorValues_Stoch(t *testing.T) {
	closes, opens, highs, lows, volumes := makeOHLCV(30)
	spec := IndicatorSpec{Alias: "my_stoch", Kind: "stoch", Params: map[string]any{}}
	curr, _, err := ComputeIndicatorValues(spec, closes, opens, highs, lows, volumes)
	if err != nil {
		t.Fatalf("stoch error: %v", err)
	}
	if _, ok := curr.(map[string]float64); !ok {
		t.Errorf("stoch curr type = %T, want map[string]float64", curr)
	}
}

func TestComputeIndicatorValues_VWAP(t *testing.T) {
	closes, opens, highs, lows, volumes := makeOHLCV(30)
	spec := IndicatorSpec{Alias: "my_vwap", Kind: "vwap", Params: map[string]any{}}
	_, _, err := ComputeIndicatorValues(spec, closes, opens, highs, lows, volumes)
	if err != nil {
		t.Fatalf("vwap error: %v", err)
	}
}

func TestComputeIndicatorValues_ROC(t *testing.T) {
	closes, opens, highs, lows, volumes := makeOHLCV(20)
	spec := IndicatorSpec{Alias: "my_roc", Kind: "roc", Params: map[string]any{"period": float64(9)}}
	_, _, err := ComputeIndicatorValues(spec, closes, opens, highs, lows, volumes)
	if err != nil {
		t.Fatalf("roc error: %v", err)
	}
}

func TestValidateIndicatorSpec_Valid(t *testing.T) {
	spec := IndicatorSpec{Alias: "my_rsi", Kind: "rsi", Params: map[string]any{"period": 14}}
	if err := ValidateIndicatorSpec(spec); err != nil {
		t.Errorf("ValidateIndicatorSpec valid = %v, want nil", err)
	}
}

func TestValidateIndicatorSpec_EmptyAlias(t *testing.T) {
	spec := IndicatorSpec{Alias: "", Kind: "rsi", Params: map[string]any{}}
	if err := ValidateIndicatorSpec(spec); err == nil {
		t.Error("expected error for empty alias")
	}
}

func TestValidateIndicatorSpec_InvalidAlias_StartsWithDigit(t *testing.T) {
	spec := IndicatorSpec{Alias: "1bad", Kind: "rsi", Params: map[string]any{}}
	if err := ValidateIndicatorSpec(spec); err == nil {
		t.Error("expected error for alias starting with digit")
	}
}

func TestValidateIndicatorSpec_InvalidAlias_HasDash(t *testing.T) {
	spec := IndicatorSpec{Alias: "my-rsi", Kind: "rsi", Params: map[string]any{}}
	if err := ValidateIndicatorSpec(spec); err == nil {
		t.Error("expected error for alias with dash")
	}
}

func TestValidateIndicatorSpec_UnknownKind(t *testing.T) {
	spec := IndicatorSpec{Alias: "my_x", Kind: "unknown_kind", Params: map[string]any{}}
	if err := ValidateIndicatorSpec(spec); err == nil {
		t.Error("expected error for unknown indicator kind")
	}
}

func TestValidateIndicatorSpec_UnsupportedParam(t *testing.T) {
	spec := IndicatorSpec{Alias: "my_rsi", Kind: "rsi", Params: map[string]any{"badparam": 1}}
	if err := ValidateIndicatorSpec(spec); err == nil {
		t.Error("expected error for unsupported param")
	}
}
