package config

// Environment represents an isolated gcloud environment directory.
type Environment struct {
	Name          string   `json:"name"`
	Path          string   `json:"path"`
	ActiveConfig  string   `json:"active_config"`
	Project       string   `json:"project"`
	Account       string   `json:"account"`
	ImpersonateSA string   `json:"impersonate_service_account"`
	ADCExists     bool     `json:"adc_exists"`
	ADC           *ADCInfo `json:"adc,omitempty"`
	IsActive      bool     `json:"is_active"`
}

// CurrentState represents the actively resolved GCP environment from shell and config state.
type CurrentState struct {
	ActiveEnvName           string   `json:"active_env_name"`
	IsCustomConfig          bool     `json:"is_custom_config"`
	DerivedFromPath         bool     `json:"derived_from_path"`
	CloudSDKConfig          string   `json:"cloudsdk_config"`
	GCPEnv                  string   `json:"gcp_env"`
	ActiveConfig            string   `json:"active_config"`
	Project                 string   `json:"project"`
	Account                 string   `json:"account"`
	QuotaProject            string   `json:"quota_project"`
	ImpersonateSA           string   `json:"impersonate_service_account"`
	GoogleAppCredentials    string   `json:"google_application_credentials"`
	ADC                     *ADCInfo `json:"adc,omitempty"`
}
