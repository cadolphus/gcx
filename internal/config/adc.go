package config

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
)

// ADCInfo holds parsed details from application_default_credentials.json
type ADCInfo struct {
	Type                  string `json:"type"`
	ClientEmail           string `json:"client_email,omitempty"`
	ImpersonationURL      string `json:"service_account_impersonation_url,omitempty"`
	ImpersonatedEmail     string `json:"impersonated_email,omitempty"`
	QuotaProjectID        string `json:"quota_project_id,omitempty"`
	SourceCredentialsType string `json:"source_credentials_type,omitempty"`
}

var impersonationURLRegex = regexp.MustCompile(`/serviceAccounts/([^:/]+@[^:/]+)`)

// ParseADC reads and parses an application_default_credentials.json file.
func ParseADC(filePath string) (*ADCInfo, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read ADC file: %w", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse ADC JSON: %w", err)
	}

	info := &ADCInfo{}
	if t, ok := raw["type"].(string); ok {
		info.Type = t
	}
	if ce, ok := raw["client_email"].(string); ok {
		info.ClientEmail = ce
	}
	if qp, ok := raw["quota_project_id"].(string); ok {
		info.QuotaProjectID = qp
	}
	if impURL, ok := raw["service_account_impersonation_url"].(string); ok {
		info.ImpersonationURL = impURL
		matches := impersonationURLRegex.FindStringSubmatch(impURL)
		if len(matches) > 1 {
			info.ImpersonatedEmail = matches[1]
		}
	}
	if sc, ok := raw["source_credentials"].(map[string]interface{}); ok {
		if sct, ok := sc["type"].(string); ok {
			info.SourceCredentialsType = sct
		}
	}

	return info, nil
}
