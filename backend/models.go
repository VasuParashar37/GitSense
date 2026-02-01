package main

type ProjectSummary struct {
	TotalFiles    int     `json:"total_files"`
	ActiveFiles   int     `json:"active_files"`
	StableFiles   int     `json:"stable_files"`
	InactiveFiles int     `json:"inactive_files"`
	ActivityScore float64 `json:"activity_score"`
	ProjectState  string  `json:"project_state"`
}
