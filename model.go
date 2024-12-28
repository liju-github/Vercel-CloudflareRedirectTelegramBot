package main

import "time"

const vercelAPIURL = "https://api.vercel.com"
type RedirectRule struct {
	ID               string    `json:"id"`
	Version          string    `json:"version"`
	Action           string    `json:"action"`
	Expression       string    `json:"expression"`
	Description      string    `json:"description"`
	LastUpdated      time.Time `json:"last_updated"`
	Ref              string    `json:"ref"`
	Enabled          bool      `json:"enabled"`
	ActionParameters struct {
		FromValue struct {
			StatusCode int `json:"status_code"`
			TargetURL  struct {
				Value string `json:"value"`
			} `json:"target_url"`
			PreserveQueryString bool `json:"preserve_query_string"`
		} `json:"from_value"`
	} `json:"action_parameters"`
}

type RedirectRulesResponse struct {
	Result struct {
		ID    string         `json:"id"`
		Rules []RedirectRule `json:"rules"`
	} `json:"result"`
	Success bool `json:"success"`
}
