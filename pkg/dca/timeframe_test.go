package dca

import (
	"testing"
)

// TestTimeframeToCron tests the TimeframeToCron function
func TestTimeframeToCron(t *testing.T) {
	tests := []struct {
		name    string
		tf      string
		wantCron string
		wantErr bool
	}{
		{
			name:    "1m",
			tf:      "1m",
			wantCron: "* * * * *",
			wantErr: false,
		},
		{
			name:    "5m",
			tf:      "5m",
			wantCron: "*/5 * * * *",
			wantErr: false,
		},
		{
			name:    "15m",
			tf:      "15m",
			wantCron: "*/15 * * * *",
			wantErr: false,
		},
		{
			name:    "30m",
			tf:      "30m",
			wantCron: "*/30 * * * *",
			wantErr: false,
		},
		{
			name:    "1h",
			tf:      "1h",
			wantCron: "0 * * * *",
			wantErr: false,
		},
		{
			name:    "2h",
			tf:      "2h",
			wantCron: "0 */2 * * *",
			wantErr: false,
		},
		{
			name:    "4h",
			tf:      "4h",
			wantCron: "0 */4 * * *",
			wantErr: false,
		},
		{
			name:    "6h",
			tf:      "6h",
			wantCron: "0 */6 * * *",
			wantErr: false,
		},
		{
			name:    "12h",
			tf:      "12h",
			wantCron: "0 */12 * * *",
			wantErr: false,
		},
		{
			name:    "1d",
			tf:      "1d",
			wantCron: "0 0 * * *",
			wantErr: false,
		},
		{
			name:    "1w",
			tf:      "1w",
			wantCron: "0 0 * * 1",
			wantErr: false,
		},
		{
			name:    "unsupported",
			tf:      "3h",
			wantCron: "",
			wantErr: true,
		},
		{
			name:    "empty string",
			tf:      "",
			wantCron: "",
			wantErr: true,
		},
		{
			name:    "invalid format",
			tf:      "invalid",
			wantCron: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TimeframeToCron(tt.tf)

			if tt.wantErr && err == nil {
				t.Fatalf("TimeframeToCron(%q) expected error, got nil", tt.tf)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("TimeframeToCron(%q) unexpected error: %v", tt.tf, err)
			}

			if got != tt.wantCron {
				t.Errorf("TimeframeToCron(%q) = %q, want %q", tt.tf, got, tt.wantCron)
			}
		})
	}
}
