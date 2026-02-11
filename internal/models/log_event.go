package models

type LogEvent struct {
	Session      string  `json:"session"`
	Action       string  `json:"action"`
	Statement    string  `json:"statement,omitempty"`
	Count        int     `json:"count"`
	TotalDeltaMS float64 `json:"total_delta_ms"`
	AvgDeltaMS   float64 `json:"avg_delta_ms"`
}