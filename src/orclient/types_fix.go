package orclient

// CostEstimate contains cost information for a request
type CostEstimate struct {
	PromptCost     float64 `json:"prompt_cost"`
	CompletionCost float64 `json:"completion_cost"`
	TotalCost      float64 `json:"total_cost"`
	Currency       string  `json:"currency"`
}
