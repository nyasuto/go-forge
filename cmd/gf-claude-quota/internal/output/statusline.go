package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"gf-claude-quota/internal/api"
)

// StatusLineInput represents the JSON input from Claude Code's statusLine.
type StatusLineInput struct {
	Model         string  `json:"model"`
	ContextWindow float64 `json:"context_window"`
	ContextUsed   float64 `json:"context_used"`
	Cost          float64 `json:"cost"`
}

// FormatStatusLine reads optional JSON from stdin, merges with quota data,
// and writes a single-line output suitable for Claude Code's statusLine.
// Format: ⚡5h:42%(2h14m) 📅7d:18% | {model} | ctx:{ctx_pct}% | ${cost}
func FormatStatusLine(w io.Writer, usage *api.UsageResponse, stdinData []byte) {
	var input *StatusLineInput
	if len(stdinData) > 0 {
		trimmed := strings.TrimSpace(string(stdinData))
		if trimmed != "" {
			var si StatusLineInput
			if err := json.Unmarshal([]byte(trimmed), &si); err == nil {
				input = &si
			}
		}
	}

	parts := []string{}

	// Quota section
	quotaParts := buildQuotaParts(usage)
	if len(quotaParts) > 0 {
		parts = append(parts, strings.Join(quotaParts, " "))
	}

	// StatusLine input sections
	if input != nil {
		if input.Model != "" {
			parts = append(parts, input.Model)
		}
		if input.ContextWindow > 0 {
			ctxPct := input.ContextUsed / input.ContextWindow * 100
			parts = append(parts, fmt.Sprintf("ctx:%.0f%%", ctxPct))
		}
		if input.Cost > 0 {
			parts = append(parts, fmt.Sprintf("$%.2f", input.Cost))
		}
	}

	if len(parts) > 0 {
		fmt.Fprintln(w, strings.Join(parts, " | "))
	}
}

func buildQuotaParts(usage *api.UsageResponse) []string {
	parts := []string{}
	if usage.FiveHour != nil {
		s := fmt.Sprintf("⚡5h:%.0f%%", usage.FiveHour.Utilization)
		if usage.FiveHour.ResetsAt != nil {
			s += fmt.Sprintf("(%s)", FormatResetTime(*usage.FiveHour.ResetsAt))
		}
		parts = append(parts, s)
	}
	if usage.SevenDay != nil {
		s := fmt.Sprintf("📅7d:%.0f%%", usage.SevenDay.Utilization)
		if usage.SevenDay.ResetsAt != nil {
			s += fmt.Sprintf("(%s)", FormatResetTime(*usage.SevenDay.ResetsAt))
		}
		parts = append(parts, s)
	}
	if usage.SevenDayOpus != nil && usage.SevenDayOpus.Utilization > 0 {
		s := fmt.Sprintf("opus:%.0f%%", usage.SevenDayOpus.Utilization)
		if usage.SevenDayOpus.ResetsAt != nil {
			s += fmt.Sprintf("(%s)", FormatResetTime(*usage.SevenDayOpus.ResetsAt))
		}
		parts = append(parts, s)
	}
	return parts
}

// FormatTemplate applies a user-defined template string with variable substitution.
// Supported variables: {5h}, {5h_bar}, {5h_reset}, {7d}, {7d_bar}, {7d_reset},
// {opus}, {model}, {ctx_pct}, {cost}
func FormatTemplate(w io.Writer, usage *api.UsageResponse, stdinData []byte, tmpl string) {
	var input *StatusLineInput
	if len(stdinData) > 0 {
		trimmed := strings.TrimSpace(string(stdinData))
		if trimmed != "" {
			var si StatusLineInput
			if err := json.Unmarshal([]byte(trimmed), &si); err == nil {
				input = &si
			}
		}
	}

	vars := buildTemplateVars(usage, input)
	result := tmpl
	for k, v := range vars {
		result = strings.ReplaceAll(result, "{"+k+"}", v)
	}
	fmt.Fprintln(w, result)
}

func buildTemplateVars(usage *api.UsageResponse, input *StatusLineInput) map[string]string {
	vars := map[string]string{
		"5h":       "N/A",
		"5h_bar":   BuildBar(0, 10),
		"5h_reset": "",
		"7d":       "N/A",
		"7d_bar":   BuildBar(0, 10),
		"7d_reset": "",
		"opus":     "N/A",
		"model":    "",
		"ctx_pct":  "",
		"cost":     "",
	}

	if usage.FiveHour != nil {
		vars["5h"] = fmt.Sprintf("%.0f%%", usage.FiveHour.Utilization)
		vars["5h_bar"] = BuildBar(usage.FiveHour.Utilization, 10)
		if usage.FiveHour.ResetsAt != nil {
			vars["5h_reset"] = FormatResetTime(*usage.FiveHour.ResetsAt)
		}
	}
	if usage.SevenDay != nil {
		vars["7d"] = fmt.Sprintf("%.0f%%", usage.SevenDay.Utilization)
		vars["7d_bar"] = BuildBar(usage.SevenDay.Utilization, 10)
		if usage.SevenDay.ResetsAt != nil {
			vars["7d_reset"] = FormatResetTime(*usage.SevenDay.ResetsAt)
		}
	}
	if usage.SevenDayOpus != nil {
		vars["opus"] = fmt.Sprintf("%.0f%%", usage.SevenDayOpus.Utilization)
	}

	if input != nil {
		if input.Model != "" {
			vars["model"] = input.Model
		}
		if input.ContextWindow > 0 {
			vars["ctx_pct"] = fmt.Sprintf("%.0f", input.ContextUsed/input.ContextWindow*100)
		}
		if input.Cost > 0 {
			vars["cost"] = fmt.Sprintf("%.2f", input.Cost)
		}
	}

	return vars
}
