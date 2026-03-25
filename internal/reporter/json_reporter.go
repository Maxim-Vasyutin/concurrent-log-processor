package reporter

import (
	"encoding/json"
	"fmt"
	"os"
)

type FailedRequestReport struct {
	RequestID string   `json:"request_id"`
	Timeline  []string `json:"timeline"`
}

type AnalysisResult struct {
	FailedRequests []FailedRequestReport `json:"failed_requests"`
}

type JSONReporter struct{}

func WriteJSONReport(result AnalysisResult, filename string) error {
	return JSONReporter{}.Write(result, filename)
}

func (r JSONReporter) Write(result AnalysisResult, filename string) error {
	payload, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json report: %w", err)
	}

	if err := os.WriteFile(filename, payload, 0o644); err != nil {
		return fmt.Errorf("write json report to %q: %w", filename, err)
	}

	return nil
}
