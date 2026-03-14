package api

// UsageResponse represents the response from the Anthropic usage API.
type UsageResponse struct {
	FiveHour      *UsageWindow `json:"five_hour"`
	SevenDay      *UsageWindow `json:"seven_day"`
	SevenDayOAuth *UsageWindow `json:"seven_day_oauth_apps"`
	SevenDayOpus  *UsageWindow `json:"seven_day_opus"`
}

// UsageWindow represents a single usage window with utilization percentage and reset time.
type UsageWindow struct {
	Utilization float64 `json:"utilization"`
	ResetsAt    *string `json:"resets_at"`
}
