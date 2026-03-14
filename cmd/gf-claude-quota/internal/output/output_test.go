package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"gf-claude-quota/internal/api"
)

// --- BuildBar tests ---

func TestBuildBar(t *testing.T) {
	tests := []struct {
		name  string
		pct   float64
		width int
		want  string
	}{
		{"0%", 0, 10, "░░░░░░░░░░"},
		{"50%", 50, 10, "█████░░░░░"},
		{"100%", 100, 10, "██████████"},
		{"42%", 42, 10, "████░░░░░░"},
		{"99.5%", 99.5, 10, "██████████"},
		{"negative", -5, 10, "░░░░░░░░░░"},
		{"over 100", 150, 10, "██████████"},
		{"width 20", 50, 20, "██████████░░░░░░░░░░"},
		{"1%", 1, 10, "░░░░░░░░░░"},
		{"5% rounds to 1", 5, 10, "█░░░░░░░░░"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildBar(tt.pct, tt.width)
			if got != tt.want {
				t.Errorf("BuildBar(%v, %d) = %q, want %q", tt.pct, tt.width, got, tt.want)
			}
		})
	}
}

// --- FormatResetTime tests ---

func TestFormatResetTime(t *testing.T) {
	origNow := NowFunc
	defer func() { NowFunc = origNow }()

	tests := []struct {
		name    string
		now     time.Time
		resetAt string
		want    string
	}{
		{
			name:    "hours and minutes",
			now:     time.Date(2025, 11, 4, 2, 30, 0, 0, time.UTC),
			resetAt: "2025-11-04T04:59:59+00:00",
			want:    "2h29m",
		},
		{
			name:    "days and hours",
			now:     time.Date(2025, 11, 1, 12, 0, 0, 0, time.UTC),
			resetAt: "2025-11-06T03:59:59+00:00",
			want:    "4d15h",
		},
		{
			name:    "minutes only",
			now:     time.Date(2025, 11, 4, 4, 45, 0, 0, time.UTC),
			resetAt: "2025-11-04T04:59:59+00:00",
			want:    "14m",
		},
		{
			name:    "already passed",
			now:     time.Date(2025, 11, 5, 0, 0, 0, 0, time.UTC),
			resetAt: "2025-11-04T04:59:59+00:00",
			want:    "now",
		},
		{
			name:    "invalid time format",
			now:     time.Date(2025, 11, 4, 0, 0, 0, 0, time.UTC),
			resetAt: "not-a-time",
			want:    "not-a-time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			NowFunc = func() time.Time { return tt.now }
			got := FormatResetTime(tt.resetAt)
			if got != tt.want {
				t.Errorf("FormatResetTime(%q) = %q, want %q", tt.resetAt, got, tt.want)
			}
		})
	}
}

// --- ColorLevel tests ---

func TestColorLevel(t *testing.T) {
	tests := []struct {
		name string
		pct  float64
		want string
	}{
		{"0% is green", 0, "green"},
		{"49% is green", 49, "green"},
		{"50% is yellow", 50, "yellow"},
		{"79% is yellow", 79, "yellow"},
		{"80% is red", 80, "red"},
		{"100% is red", 100, "red"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ColorLevel(tt.pct)
			if got != tt.want {
				t.Errorf("ColorLevel(%v) = %q, want %q", tt.pct, got, tt.want)
			}
		})
	}
}

// --- Colorize tests ---

func TestColorize(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		pct      float64
		wantCode string
	}{
		{"green", "42%", 42, "\033[32m"},
		{"yellow", "60%", 60, "\033[33m"},
		{"red", "90%", 90, "\033[31m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Colorize(tt.text, tt.pct)
			if !strings.HasPrefix(got, tt.wantCode) {
				t.Errorf("Colorize(%q, %v) = %q, want prefix %q", tt.text, tt.pct, got, tt.wantCode)
			}
			if !strings.HasSuffix(got, ansiReset) {
				t.Errorf("Colorize(%q, %v) = %q, want suffix %q", tt.text, tt.pct, got, ansiReset)
			}
			if !strings.Contains(got, tt.text) {
				t.Errorf("Colorize(%q, %v) = %q, should contain original text", tt.text, tt.pct, got)
			}
		})
	}
}

// --- ParseColorMode tests ---

func TestParseColorMode(t *testing.T) {
	tests := []struct {
		input   string
		want    ColorMode
		wantErr bool
	}{
		{"auto", ColorAuto, false},
		{"always", ColorAlways, false},
		{"never", ColorNever, false},
		{"invalid", ColorAuto, true},
		{"", ColorAuto, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseColorMode(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseColorMode(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseColorMode(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// --- FormatText tests ---

func sampleUsage() *api.UsageResponse {
	resetAt5h := "2025-11-04T04:59:59+00:00"
	resetAt7d := "2025-11-06T03:59:59+00:00"
	return &api.UsageResponse{
		FiveHour: &api.UsageWindow{
			Utilization: 42.0,
			ResetsAt:    &resetAt5h,
		},
		SevenDay: &api.UsageWindow{
			Utilization: 18.0,
			ResetsAt:    &resetAt7d,
		},
		SevenDayOpus: &api.UsageWindow{
			Utilization: 0.0,
			ResetsAt:    nil,
		},
	}
}

func TestFormatText(t *testing.T) {
	origNow := NowFunc
	defer func() { NowFunc = origNow }()
	NowFunc = func() time.Time {
		return time.Date(2025, 11, 4, 2, 30, 0, 0, time.UTC)
	}

	var buf bytes.Buffer
	FormatText(&buf, sampleUsage(), false)
	out := buf.String()

	// Check header
	if !strings.Contains(out, "Claude Code Usage") {
		t.Error("missing header")
	}
	// Check 5h window
	if !strings.Contains(out, "5h Session") {
		t.Error("missing 5h Session label")
	}
	if !strings.Contains(out, "42%") {
		t.Error("missing 42% utilization")
	}
	if !strings.Contains(out, "resets in 2h29m") {
		t.Error("missing reset time")
	}
	// Check 7d window
	if !strings.Contains(out, "7d Weekly") {
		t.Error("missing 7d Weekly label")
	}
	if !strings.Contains(out, "18%") {
		t.Error("missing 18% utilization")
	}
	// Check opus window
	if !strings.Contains(out, "7d Opus") {
		t.Error("missing 7d Opus label")
	}
	if !strings.Contains(out, "0%") {
		t.Error("missing 0% utilization")
	}
}

func TestFormatText_WithColor(t *testing.T) {
	origNow := NowFunc
	defer func() { NowFunc = origNow }()
	NowFunc = func() time.Time {
		return time.Date(2025, 11, 4, 2, 30, 0, 0, time.UTC)
	}

	var buf bytes.Buffer
	FormatText(&buf, sampleUsage(), true)
	out := buf.String()

	// Should contain ANSI color codes
	if !strings.Contains(out, "\033[") {
		t.Error("expected ANSI color codes in output")
	}
	if !strings.Contains(out, ansiReset) {
		t.Error("expected ANSI reset codes in output")
	}
}

func TestFormatText_NoColor(t *testing.T) {
	origNow := NowFunc
	defer func() { NowFunc = origNow }()
	NowFunc = func() time.Time {
		return time.Date(2025, 11, 4, 2, 30, 0, 0, time.UTC)
	}

	var buf bytes.Buffer
	FormatText(&buf, sampleUsage(), false)
	out := buf.String()

	// Should NOT contain ANSI color codes
	if strings.Contains(out, "\033[") {
		t.Error("unexpected ANSI color codes in output")
	}
}

func TestFormatText_NilWindows(t *testing.T) {
	var buf bytes.Buffer
	usage := &api.UsageResponse{}
	FormatText(&buf, usage, false)
	out := buf.String()

	if !strings.Contains(out, "Claude Code Usage") {
		t.Error("missing header even with nil windows")
	}
	if strings.Contains(out, "5h Session") {
		t.Error("should not show 5h Session when nil")
	}
}

func TestFormatText_HighUtilization(t *testing.T) {
	origNow := NowFunc
	defer func() { NowFunc = origNow }()
	NowFunc = func() time.Time {
		return time.Date(2025, 11, 4, 2, 30, 0, 0, time.UTC)
	}

	resetAt := "2025-11-04T04:59:59+00:00"
	usage := &api.UsageResponse{
		FiveHour: &api.UsageWindow{
			Utilization: 95.0,
			ResetsAt:    &resetAt,
		},
	}

	var buf bytes.Buffer
	FormatText(&buf, usage, true)
	out := buf.String()

	// High utilization should use red color
	if !strings.Contains(out, ansiRed) {
		t.Error("expected red color for high utilization")
	}
}

func TestFormatText_MediumUtilization(t *testing.T) {
	origNow := NowFunc
	defer func() { NowFunc = origNow }()
	NowFunc = func() time.Time {
		return time.Date(2025, 11, 4, 2, 30, 0, 0, time.UTC)
	}

	resetAt := "2025-11-04T04:59:59+00:00"
	usage := &api.UsageResponse{
		FiveHour: &api.UsageWindow{
			Utilization: 65.0,
			ResetsAt:    &resetAt,
		},
	}

	var buf bytes.Buffer
	FormatText(&buf, usage, true)
	out := buf.String()

	// Medium utilization should use yellow color
	if !strings.Contains(out, ansiYellow) {
		t.Error("expected yellow color for medium utilization")
	}
}

// --- FormatJSON tests ---

func TestFormatJSON(t *testing.T) {
	origNow := NowFunc
	defer func() { NowFunc = origNow }()
	NowFunc = func() time.Time {
		return time.Date(2025, 11, 4, 2, 30, 0, 0, time.UTC)
	}

	var buf bytes.Buffer
	err := FormatJSON(&buf, sampleUsage())
	if err != nil {
		t.Fatalf("FormatJSON error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	// Check five_hour
	fh, ok := result["five_hour"].(map[string]interface{})
	if !ok {
		t.Fatal("missing five_hour in JSON output")
	}
	if fh["utilization"].(float64) != 42.0 {
		t.Errorf("five_hour.utilization = %v, want 42.0", fh["utilization"])
	}
	if fh["resets_in"].(string) != "2h29m" {
		t.Errorf("five_hour.resets_in = %v, want 2h29m", fh["resets_in"])
	}

	// Check seven_day
	sd, ok := result["seven_day"].(map[string]interface{})
	if !ok {
		t.Fatal("missing seven_day in JSON output")
	}
	if sd["utilization"].(float64) != 18.0 {
		t.Errorf("seven_day.utilization = %v, want 18.0", sd["utilization"])
	}

	// Check seven_day_opus
	opus, ok := result["seven_day_opus"].(map[string]interface{})
	if !ok {
		t.Fatal("missing seven_day_opus in JSON output")
	}
	if opus["utilization"].(float64) != 0.0 {
		t.Errorf("seven_day_opus.utilization = %v, want 0.0", opus["utilization"])
	}
	// resets_in should be empty (no resets_at)
	if _, exists := opus["resets_in"]; exists {
		t.Error("seven_day_opus should not have resets_in when resets_at is nil")
	}
}

func TestFormatJSON_NilWindows(t *testing.T) {
	var buf bytes.Buffer
	err := FormatJSON(&buf, &api.UsageResponse{})
	if err != nil {
		t.Fatalf("FormatJSON error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	// All should be omitted
	if _, exists := result["five_hour"]; exists {
		t.Error("five_hour should be omitted when nil")
	}
}

func TestFormatJSON_ValidJSON(t *testing.T) {
	origNow := NowFunc
	defer func() { NowFunc = origNow }()
	NowFunc = func() time.Time {
		return time.Date(2025, 11, 4, 2, 30, 0, 0, time.UTC)
	}

	var buf bytes.Buffer
	_ = FormatJSON(&buf, sampleUsage())

	// Verify it's valid, indented JSON
	out := buf.String()
	if !strings.Contains(out, "  ") {
		t.Error("expected indented JSON")
	}
	if !strings.HasSuffix(strings.TrimSpace(out), "}") {
		t.Error("expected JSON to end with }")
	}
}

// --- FormatOneline tests ---

func TestFormatOneline(t *testing.T) {
	origNow := NowFunc
	defer func() { NowFunc = origNow }()
	NowFunc = func() time.Time {
		return time.Date(2025, 11, 4, 2, 30, 0, 0, time.UTC)
	}

	var buf bytes.Buffer
	FormatOneline(&buf, sampleUsage())
	out := strings.TrimSpace(buf.String())

	// Should contain 5h and 7d but not opus (0%)
	if !strings.Contains(out, "5h:42%(2h29m)") {
		t.Errorf("output = %q, want to contain '5h:42%%(2h29m)'", out)
	}
	if !strings.Contains(out, "7d:18%") {
		t.Errorf("output = %q, want to contain '7d:18%%'", out)
	}
	if strings.Contains(out, "opus") {
		t.Errorf("output = %q, should not contain opus at 0%%", out)
	}
}

func TestFormatOneline_WithOpus(t *testing.T) {
	origNow := NowFunc
	defer func() { NowFunc = origNow }()
	NowFunc = func() time.Time {
		return time.Date(2025, 11, 4, 2, 30, 0, 0, time.UTC)
	}

	resetAt := "2025-11-04T04:59:59+00:00"
	usage := &api.UsageResponse{
		FiveHour: &api.UsageWindow{
			Utilization: 42.0,
			ResetsAt:    &resetAt,
		},
		SevenDayOpus: &api.UsageWindow{
			Utilization: 15.0,
			ResetsAt:    &resetAt,
		},
	}

	var buf bytes.Buffer
	FormatOneline(&buf, usage)
	out := strings.TrimSpace(buf.String())

	if !strings.Contains(out, "opus:15%") {
		t.Errorf("output = %q, want to contain 'opus:15%%'", out)
	}
}

func TestFormatOneline_NilWindows(t *testing.T) {
	var buf bytes.Buffer
	FormatOneline(&buf, &api.UsageResponse{})
	out := strings.TrimSpace(buf.String())

	if out != "" {
		t.Errorf("output = %q, want empty string for nil windows", out)
	}
}

func TestFormatOneline_NoResetTime(t *testing.T) {
	usage := &api.UsageResponse{
		FiveHour: &api.UsageWindow{
			Utilization: 42.0,
			ResetsAt:    nil,
		},
	}

	var buf bytes.Buffer
	FormatOneline(&buf, usage)
	out := strings.TrimSpace(buf.String())

	if out != "5h:42%" {
		t.Errorf("output = %q, want '5h:42%%'", out)
	}
}
