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

// --- FormatStatusLine tests ---

func TestFormatStatusLine_WithStdinData(t *testing.T) {
	origNow := NowFunc
	defer func() { NowFunc = origNow }()
	NowFunc = func() time.Time {
		return time.Date(2025, 11, 4, 2, 30, 0, 0, time.UTC)
	}

	stdinData := []byte(`{"model":"claude-sonnet-4-20250514","context_window":200000,"context_used":50000,"cost":1.23}`)

	var buf bytes.Buffer
	FormatStatusLine(&buf, sampleUsage(), stdinData)
	out := strings.TrimSpace(buf.String())

	if !strings.Contains(out, "⚡5h:42%(2h29m)") {
		t.Errorf("output = %q, want to contain '⚡5h:42%%(2h29m)'", out)
	}
	if !strings.Contains(out, "📅7d:18%") {
		t.Errorf("output = %q, want to contain '📅7d:18%%'", out)
	}
	if !strings.Contains(out, "claude-sonnet-4-20250514") {
		t.Errorf("output = %q, want to contain model name", out)
	}
	if !strings.Contains(out, "ctx:25%") {
		t.Errorf("output = %q, want to contain 'ctx:25%%'", out)
	}
	if !strings.Contains(out, "$1.23") {
		t.Errorf("output = %q, want to contain '$1.23'", out)
	}
	// Sections should be pipe-separated
	if !strings.Contains(out, " | ") {
		t.Errorf("output = %q, want pipe separators", out)
	}
}

func TestFormatStatusLine_NoStdinData(t *testing.T) {
	origNow := NowFunc
	defer func() { NowFunc = origNow }()
	NowFunc = func() time.Time {
		return time.Date(2025, 11, 4, 2, 30, 0, 0, time.UTC)
	}

	var buf bytes.Buffer
	FormatStatusLine(&buf, sampleUsage(), nil)
	out := strings.TrimSpace(buf.String())

	// Should contain quota info but no model/ctx/cost
	if !strings.Contains(out, "⚡5h:42%") {
		t.Errorf("output = %q, want to contain '⚡5h:42%%'", out)
	}
	if strings.Contains(out, " | ") {
		t.Errorf("output = %q, should not contain pipe separators without stdin data", out)
	}
}

func TestFormatStatusLine_EmptyStdin(t *testing.T) {
	var buf bytes.Buffer
	FormatStatusLine(&buf, sampleUsage(), []byte(""))
	out := strings.TrimSpace(buf.String())

	// Should still output quota info
	if !strings.Contains(out, "⚡5h:42%") {
		t.Errorf("output = %q, want to contain quota info", out)
	}
}

func TestFormatStatusLine_InvalidStdinJSON(t *testing.T) {
	var buf bytes.Buffer
	FormatStatusLine(&buf, sampleUsage(), []byte("not json"))
	out := strings.TrimSpace(buf.String())

	// Should still output quota info, ignoring invalid JSON
	if !strings.Contains(out, "⚡5h:42%") {
		t.Errorf("output = %q, want to contain quota info despite invalid stdin", out)
	}
}

func TestFormatStatusLine_NilWindows(t *testing.T) {
	stdinData := []byte(`{"model":"opus","context_window":100000,"context_used":25000,"cost":0.5}`)

	var buf bytes.Buffer
	FormatStatusLine(&buf, &api.UsageResponse{}, stdinData)
	out := strings.TrimSpace(buf.String())

	// Should show model info even without quota
	if !strings.Contains(out, "opus") {
		t.Errorf("output = %q, want to contain model name", out)
	}
	if !strings.Contains(out, "ctx:25%") {
		t.Errorf("output = %q, want to contain ctx percentage", out)
	}
}

func TestFormatStatusLine_WithOpus(t *testing.T) {
	origNow := NowFunc
	defer func() { NowFunc = origNow }()
	NowFunc = func() time.Time {
		return time.Date(2025, 11, 4, 2, 30, 0, 0, time.UTC)
	}

	resetAt := "2025-11-04T04:59:59+00:00"
	usage := &api.UsageResponse{
		FiveHour: &api.UsageWindow{Utilization: 42.0, ResetsAt: &resetAt},
		SevenDayOpus: &api.UsageWindow{Utilization: 15.0, ResetsAt: &resetAt},
	}

	var buf bytes.Buffer
	FormatStatusLine(&buf, usage, nil)
	out := strings.TrimSpace(buf.String())

	if !strings.Contains(out, "opus:15%") {
		t.Errorf("output = %q, want to contain 'opus:15%%'", out)
	}
}

func TestFormatStatusLine_ZeroCostNotShown(t *testing.T) {
	stdinData := []byte(`{"model":"sonnet","context_window":200000,"context_used":100000,"cost":0}`)

	var buf bytes.Buffer
	FormatStatusLine(&buf, sampleUsage(), stdinData)
	out := strings.TrimSpace(buf.String())

	if strings.Contains(out, "$0") {
		t.Errorf("output = %q, should not contain $0 cost", out)
	}
}

// --- FormatTemplate tests ---

func TestFormatTemplate_BasicVars(t *testing.T) {
	origNow := NowFunc
	defer func() { NowFunc = origNow }()
	NowFunc = func() time.Time {
		return time.Date(2025, 11, 4, 2, 30, 0, 0, time.UTC)
	}

	var buf bytes.Buffer
	FormatTemplate(&buf, sampleUsage(), nil, "5h:{5h} 7d:{7d}")
	out := strings.TrimSpace(buf.String())

	if out != "5h:42% 7d:18%" {
		t.Errorf("output = %q, want '5h:42%% 7d:18%%'", out)
	}
}

func TestFormatTemplate_WithStdinVars(t *testing.T) {
	stdinData := []byte(`{"model":"claude-opus","context_window":200000,"context_used":150000,"cost":2.50}`)

	var buf bytes.Buffer
	FormatTemplate(&buf, sampleUsage(), stdinData, "{5h} | {model} | ctx:{ctx_pct}% | ${cost}")
	out := strings.TrimSpace(buf.String())

	if !strings.Contains(out, "42%") {
		t.Errorf("output = %q, want to contain '42%%'", out)
	}
	if !strings.Contains(out, "claude-opus") {
		t.Errorf("output = %q, want to contain 'claude-opus'", out)
	}
	if !strings.Contains(out, "ctx:75%") {
		t.Errorf("output = %q, want to contain 'ctx:75%%'", out)
	}
	if !strings.Contains(out, "$2.50") {
		t.Errorf("output = %q, want to contain '$2.50'", out)
	}
}

func TestFormatTemplate_ResetTimeVar(t *testing.T) {
	origNow := NowFunc
	defer func() { NowFunc = origNow }()
	NowFunc = func() time.Time {
		return time.Date(2025, 11, 4, 2, 30, 0, 0, time.UTC)
	}

	var buf bytes.Buffer
	FormatTemplate(&buf, sampleUsage(), nil, "{5h}({5h_reset})")
	out := strings.TrimSpace(buf.String())

	if out != "42%(2h29m)" {
		t.Errorf("output = %q, want '42%%(2h29m)'", out)
	}
}

func TestFormatTemplate_BarVar(t *testing.T) {
	var buf bytes.Buffer
	FormatTemplate(&buf, sampleUsage(), nil, "[{5h_bar}]")
	out := strings.TrimSpace(buf.String())

	expected := "[" + BuildBar(42.0, 10) + "]"
	if out != expected {
		t.Errorf("output = %q, want %q", out, expected)
	}
}

func TestFormatTemplate_OpusVar(t *testing.T) {
	var buf bytes.Buffer
	FormatTemplate(&buf, sampleUsage(), nil, "opus:{opus}")
	out := strings.TrimSpace(buf.String())

	if out != "opus:0%" {
		t.Errorf("output = %q, want 'opus:0%%'", out)
	}
}

func TestFormatTemplate_NilWindows(t *testing.T) {
	var buf bytes.Buffer
	FormatTemplate(&buf, &api.UsageResponse{}, nil, "{5h} {7d}")
	out := strings.TrimSpace(buf.String())

	if out != "N/A N/A" {
		t.Errorf("output = %q, want 'N/A N/A'", out)
	}
}

func TestFormatTemplate_NoStdinVars(t *testing.T) {
	var buf bytes.Buffer
	FormatTemplate(&buf, sampleUsage(), nil, "m:{model} ctx:{ctx_pct}")
	out := strings.TrimSpace(buf.String())

	// model and ctx_pct should be empty strings
	if out != "m: ctx:" {
		t.Errorf("output = %q, want 'm: ctx:'", out)
	}
}

// --- buildTemplateVars tests ---

func TestBuildTemplateVars(t *testing.T) {
	origNow := NowFunc
	defer func() { NowFunc = origNow }()
	NowFunc = func() time.Time {
		return time.Date(2025, 11, 4, 2, 30, 0, 0, time.UTC)
	}

	input := &StatusLineInput{
		Model:         "sonnet",
		ContextWindow: 100000,
		ContextUsed:   50000,
		Cost:          1.5,
	}

	vars := buildTemplateVars(sampleUsage(), input)

	if vars["5h"] != "42%" {
		t.Errorf("5h = %q, want '42%%'", vars["5h"])
	}
	if vars["7d"] != "18%" {
		t.Errorf("7d = %q, want '18%%'", vars["7d"])
	}
	if vars["5h_reset"] != "2h29m" {
		t.Errorf("5h_reset = %q, want '2h29m'", vars["5h_reset"])
	}
	if vars["model"] != "sonnet" {
		t.Errorf("model = %q, want 'sonnet'", vars["model"])
	}
	if vars["ctx_pct"] != "50" {
		t.Errorf("ctx_pct = %q, want '50'", vars["ctx_pct"])
	}
	if vars["cost"] != "1.50" {
		t.Errorf("cost = %q, want '1.50'", vars["cost"])
	}
}

func TestBuildTemplateVars_NilInput(t *testing.T) {
	vars := buildTemplateVars(sampleUsage(), nil)

	if vars["model"] != "" {
		t.Errorf("model = %q, want empty", vars["model"])
	}
	if vars["ctx_pct"] != "" {
		t.Errorf("ctx_pct = %q, want empty", vars["ctx_pct"])
	}
}

// --- StatusLineInput JSON parsing tests ---

// --- Notifier tests ---

func TestNotifier_BelowThreshold(t *testing.T) {
	origSend := sendNotificationFunc
	defer func() { sendNotificationFunc = origSend }()

	called := false
	sendNotificationFunc = func(name string, util float64) {
		called = true
	}

	n := NewNotifier(80)
	n.Check("5h Session", 42.0)

	if called {
		t.Error("notification should not fire below threshold")
	}
}

func TestNotifier_AboveThreshold(t *testing.T) {
	origSend := sendNotificationFunc
	defer func() { sendNotificationFunc = origSend }()

	var gotName string
	var gotUtil float64
	sendNotificationFunc = func(name string, util float64) {
		gotName = name
		gotUtil = util
	}

	n := NewNotifier(80)
	n.Check("5h Session", 85.0)

	if gotName != "5h Session" {
		t.Errorf("notification window name = %q, want '5h Session'", gotName)
	}
	if gotUtil != 85.0 {
		t.Errorf("notification utilization = %v, want 85.0", gotUtil)
	}
}

func TestNotifier_Deduplication(t *testing.T) {
	origSend := sendNotificationFunc
	defer func() { sendNotificationFunc = origSend }()

	callCount := 0
	sendNotificationFunc = func(name string, util float64) {
		callCount++
	}

	n := NewNotifier(80)
	n.Check("5h Session", 85.0)
	n.Check("5h Session", 90.0)
	n.Check("5h Session", 95.0)

	if callCount != 1 {
		t.Errorf("notification count = %d, want 1 (should deduplicate)", callCount)
	}
}

func TestNotifier_ResetAfterDrop(t *testing.T) {
	origSend := sendNotificationFunc
	defer func() { sendNotificationFunc = origSend }()

	callCount := 0
	sendNotificationFunc = func(name string, util float64) {
		callCount++
	}

	n := NewNotifier(80)
	n.Check("5h Session", 85.0) // fires
	n.Check("5h Session", 50.0) // drops below, resets
	n.Check("5h Session", 90.0) // fires again

	if callCount != 2 {
		t.Errorf("notification count = %d, want 2 (should fire again after drop)", callCount)
	}
}

func TestNotifier_MultipleWindows(t *testing.T) {
	origSend := sendNotificationFunc
	defer func() { sendNotificationFunc = origSend }()

	windows := []string{}
	sendNotificationFunc = func(name string, util float64) {
		windows = append(windows, name)
	}

	n := NewNotifier(50)
	n.Check("5h Session", 60.0)
	n.Check("7d Weekly", 55.0)
	n.Check("7d Opus", 30.0) // below threshold

	if len(windows) != 2 {
		t.Errorf("notification count = %d, want 2", len(windows))
	}
	if windows[0] != "5h Session" || windows[1] != "7d Weekly" {
		t.Errorf("notifications = %v, want [5h Session, 7d Weekly]", windows)
	}
}

func TestNotifier_ExactThreshold(t *testing.T) {
	origSend := sendNotificationFunc
	defer func() { sendNotificationFunc = origSend }()

	called := false
	sendNotificationFunc = func(name string, util float64) {
		called = true
	}

	n := NewNotifier(80)
	n.Check("5h Session", 80.0)

	if !called {
		t.Error("notification should fire at exact threshold")
	}
}

func TestNotifier_ZeroThreshold(t *testing.T) {
	origSend := sendNotificationFunc
	defer func() { sendNotificationFunc = origSend }()

	called := false
	sendNotificationFunc = func(name string, util float64) {
		called = true
	}

	n := NewNotifier(0)
	n.Check("5h Session", 0.0)

	if !called {
		t.Error("notification should fire at 0% with threshold 0")
	}
}

// --- FormatXbar tests ---

func TestFormatXbar_Normal(t *testing.T) {
	origNow := NowFunc
	defer func() { NowFunc = origNow }()
	NowFunc = func() time.Time {
		return time.Date(2025, 11, 4, 2, 30, 0, 0, time.UTC)
	}

	var buf bytes.Buffer
	FormatXbar(&buf, sampleUsage())
	out := buf.String()

	// Menu bar title
	if !strings.Contains(out, "\u26a142%") {
		t.Error("missing ⚡42% in title")
	}
	if !strings.Contains(out, "\U0001f4c518%") {
		t.Error("missing 📅18% in title")
	}
	if !strings.Contains(out, "| color=white sfcolor=green") {
		t.Error("missing color=white sfcolor=green in title (max 42% should be green)")
	}

	// Dropdown content
	if !strings.Contains(out, "5h Session | color=green") {
		t.Error("missing 5h Session dropdown")
	}
	if !strings.Contains(out, "7d Weekly | color=green") {
		t.Error("missing 7d Weekly dropdown")
	}
	if !strings.Contains(out, "font=Menlo") {
		t.Error("missing font=Menlo in bar line")
	}
	if !strings.Contains(out, "resets in 2h29m") {
		t.Error("missing reset time")
	}

	// Footer
	if !strings.Contains(out, "Refresh | refresh=true") {
		t.Error("missing Refresh action")
	}
	if !strings.Contains(out, "Open Claude | href=https://claude.ai") {
		t.Error("missing Open Claude link")
	}
}

func TestFormatXbar_HighUtilization(t *testing.T) {
	resetAt := "2099-01-01T00:00:00+00:00"
	usage := &api.UsageResponse{
		FiveHour: &api.UsageWindow{Utilization: 90.0, ResetsAt: &resetAt},
		SevenDay: &api.UsageWindow{Utilization: 60.0, ResetsAt: &resetAt},
	}

	var buf bytes.Buffer
	FormatXbar(&buf, usage)
	out := buf.String()

	// Max is 90% → title white with sfcolor red
	if !strings.Contains(out, "| color=white sfcolor=red") {
		t.Error("title should have color=white sfcolor=red for 90% max utilization")
	}
	if !strings.Contains(out, "5h Session | color=red") {
		t.Error("5h Session should be red at 90%")
	}
	if !strings.Contains(out, "7d Weekly | color=yellow") {
		t.Error("7d Weekly should be yellow at 60%")
	}
}

func TestFormatXbar_MediumUtilization(t *testing.T) {
	resetAt := "2099-01-01T00:00:00+00:00"
	usage := &api.UsageResponse{
		FiveHour: &api.UsageWindow{Utilization: 55.0, ResetsAt: &resetAt},
	}

	var buf bytes.Buffer
	FormatXbar(&buf, usage)
	out := buf.String()

	// Max is 55% → yellow
	lines := strings.Split(out, "\n")
	if !strings.Contains(lines[0], "| color=white sfcolor=yellow") {
		t.Errorf("title should have color=white sfcolor=yellow for 55%%, got: %s", lines[0])
	}
}

func TestFormatXbar_NilWindows(t *testing.T) {
	var buf bytes.Buffer
	FormatXbar(&buf, &api.UsageResponse{})
	out := strings.TrimSpace(buf.String())

	if out != "Claude \u23f3" {
		t.Errorf("output = %q, want 'Claude ⏳'", out)
	}
}

func TestFormatXbar_Only5h(t *testing.T) {
	resetAt := "2099-01-01T00:00:00+00:00"
	usage := &api.UsageResponse{
		FiveHour: &api.UsageWindow{Utilization: 30.0, ResetsAt: &resetAt},
	}

	var buf bytes.Buffer
	FormatXbar(&buf, usage)
	out := buf.String()

	if !strings.Contains(out, "\u26a130%") {
		t.Error("missing ⚡30% in title")
	}
	// Should NOT contain 📅 since no 7d data
	if strings.Contains(out, "\U0001f4c5") {
		t.Error("should not contain 📅 without 7d data")
	}
}

func TestFormatXbar_NoResetTime(t *testing.T) {
	usage := &api.UsageResponse{
		FiveHour: &api.UsageWindow{Utilization: 42.0, ResetsAt: nil},
	}

	var buf bytes.Buffer
	FormatXbar(&buf, usage)
	out := buf.String()

	if strings.Contains(out, "resets in") {
		t.Error("should not show reset time when ResetsAt is nil")
	}
}

func TestFormatXbar_WithOpus(t *testing.T) {
	resetAt := "2099-01-01T00:00:00+00:00"
	usage := &api.UsageResponse{
		FiveHour:     &api.UsageWindow{Utilization: 42.0, ResetsAt: &resetAt},
		SevenDay:     &api.UsageWindow{Utilization: 18.0, ResetsAt: &resetAt},
		SevenDayOpus: &api.UsageWindow{Utilization: 85.0, ResetsAt: &resetAt},
	}

	var buf bytes.Buffer
	FormatXbar(&buf, usage)
	out := buf.String()

	if !strings.Contains(out, "7d Opus | color=red") {
		t.Error("missing 7d Opus at red color")
	}
	// Max is 85% → title should be red
	lines := strings.Split(out, "\n")
	if !strings.Contains(lines[0], "| color=white sfcolor=red") {
		t.Errorf("title should have color=white sfcolor=red when opus is 85%%, got: %s", lines[0])
	}
}

func TestMaxUtilization(t *testing.T) {
	tests := []struct {
		name  string
		usage *api.UsageResponse
		want  float64
	}{
		{"all nil", &api.UsageResponse{}, 0.0},
		{"single window", &api.UsageResponse{
			FiveHour: &api.UsageWindow{Utilization: 42.0},
		}, 42.0},
		{"multiple windows", &api.UsageResponse{
			FiveHour: &api.UsageWindow{Utilization: 42.0},
			SevenDay: &api.UsageWindow{Utilization: 85.0},
		}, 85.0},
		{"opus highest", &api.UsageResponse{
			FiveHour:     &api.UsageWindow{Utilization: 10.0},
			SevenDayOpus: &api.UsageWindow{Utilization: 95.0},
		}, 95.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maxUtilization(tt.usage)
			if got != tt.want {
				t.Errorf("maxUtilization() = %v, want %v", got, tt.want)
			}
		})
	}
}

// --- ClearTerminalSeq tests ---

func TestClearTerminalSeq(t *testing.T) {
	seq := ClearTerminalSeq()
	if seq != "\033[2J\033[H" {
		t.Errorf("ClearTerminalSeq() = %q, want ANSI clear+home", seq)
	}
}

func TestStatusLineInput_Parsing(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  StatusLineInput
	}{
		{
			name:  "full input",
			input: `{"model":"claude-sonnet-4-20250514","context_window":200000,"context_used":100000,"cost":1.5}`,
			want:  StatusLineInput{Model: "claude-sonnet-4-20250514", ContextWindow: 200000, ContextUsed: 100000, Cost: 1.5},
		},
		{
			name:  "partial input",
			input: `{"model":"opus"}`,
			want:  StatusLineInput{Model: "opus"},
		},
		{
			name:  "empty object",
			input: `{}`,
			want:  StatusLineInput{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got StatusLineInput
			err := json.Unmarshal([]byte(tt.input), &got)
			if err != nil {
				t.Fatalf("unmarshal error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
