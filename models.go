package main

// PreferencesResponse is returned for full preference lookups.
type PreferencesResponse struct {
	UserID      string            `json:"userId"`
	Preferences map[string]string `json:"preferences"`
}

// SinglePrefResponse is returned for single-key lookups.
type SinglePrefResponse struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
