package output

import (
	"encoding/json"
	"fmt"
	"io"

	"gf-claude-quota/internal/api"
)

// jsonWindow is the JSON output representation of a usage window.
type jsonWindow struct {
	Utilization float64 `json:"utilization"`
	ResetsAt    *string `json:"resets_at"`
	ResetsIn    string  `json:"resets_in,omitempty"`
}

// jsonOutput is the top-level JSON output structure.
type jsonOutput struct {
	FiveHour     *jsonWindow `json:"five_hour,omitempty"`
	SevenDay     *jsonWindow `json:"seven_day,omitempty"`
	SevenDayOpus *jsonWindow `json:"seven_day_opus,omitempty"`
}

func toJSONWindow(w *api.UsageWindow) *jsonWindow {
	if w == nil {
		return nil
	}
	jw := &jsonWindow{
		Utilization: w.Utilization,
		ResetsAt:    w.ResetsAt,
	}
	if w.ResetsAt != nil {
		jw.ResetsIn = FormatResetTime(*w.ResetsAt)
	}
	return jw
}

// FormatJSON writes usage data in JSON format.
func FormatJSON(w io.Writer, usage *api.UsageResponse) error {
	out := jsonOutput{
		FiveHour:     toJSONWindow(usage.FiveHour),
		SevenDay:     toJSONWindow(usage.SevenDay),
		SevenDayOpus: toJSONWindow(usage.SevenDayOpus),
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// FormatOneline writes usage data in a compact one-line format: "5h:42% 7d:18%"
func FormatOneline(w io.Writer, usage *api.UsageResponse) {
	parts := []string{}
	if usage.FiveHour != nil {
		s := fmt.Sprintf("5h:%.0f%%", usage.FiveHour.Utilization)
		if usage.FiveHour.ResetsAt != nil {
			s += fmt.Sprintf("(%s)", FormatResetTime(*usage.FiveHour.ResetsAt))
		}
		parts = append(parts, s)
	}
	if usage.SevenDay != nil {
		s := fmt.Sprintf("7d:%.0f%%", usage.SevenDay.Utilization)
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

	for i, p := range parts {
		if i > 0 {
			fmt.Fprint(w, " ")
		}
		fmt.Fprint(w, p)
	}
	fmt.Fprintln(w)
}
