package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// GetEnvsDir returns the root directory for gcloud environments.
// Priority: GCX_HOME > GCLOUD_ENVS_DIR > ~/.gcloud-envs
func GetEnvsDir() string {
	if dir := os.Getenv("GCX_HOME"); dir != "" {
		return expandHome(dir)
	}
	if dir := os.Getenv("GCLOUD_ENVS_DIR"); dir != "" {
		return expandHome(dir)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}
	return filepath.Join(home, ".gcloud-envs")
}

// GetGlobalGCloudDir returns the default global gcloud configuration directory.
func GetGlobalGCloudDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}
	return filepath.Join(home, ".config", "gcloud")
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			if path == "~" {
				return home
			}
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

// ListEnvironments returns all available gcloud environments discovered in the envs dir.
func ListEnvironments() ([]*Environment, error) {
	envsDir := GetEnvsDir()
	entries, err := os.ReadDir(envsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Environment{}, nil
		}
		return nil, fmt.Errorf("failed to read environments directory %s: %w", envsDir, err)
	}

	activeCloudSDK := os.Getenv("CLOUDSDK_CONFIG")
	activeGCPEnv := os.Getenv("GCP_ENV")

	var envs []*Environment
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}

		env, err := GetEnvironment(name)
		if err != nil {
			continue
		}

		if (activeCloudSDK != "" && filepath.Clean(activeCloudSDK) == filepath.Clean(env.Path)) ||
			(activeGCPEnv != "" && activeGCPEnv == name) {
			env.IsActive = true
		}

		envs = append(envs, env)
	}

	sort.Slice(envs, func(i, j int) bool {
		return envs[i].Name < envs[j].Name
	})

	return envs, nil
}

// GetEnvironment inspects a specific environment directory and returns its details.
func GetEnvironment(name string) (*Environment, error) {
	envsDir := GetEnvsDir()
	dir := filepath.Join(envsDir, name)

	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("environment '%s' not found at %s", name, dir)
	}

	env := &Environment{
		Name: name,
		Path: dir,
	}

	// 1. Read active_config if present
	activeCfgName := "default"
	activeCfgBytes, err := os.ReadFile(filepath.Join(dir, "active_config"))
	if err == nil {
		trimmed := strings.TrimSpace(string(activeCfgBytes))
		if trimmed != "" {
			activeCfgName = trimmed
		}
	}
	env.ActiveConfig = activeCfgName

	// 2. Parse INI config
	confFile := filepath.Join(dir, "configurations", "config_"+activeCfgName)
	if _, err := os.Stat(confFile); os.IsNotExist(err) {
		// Fallback to config_default if specific active config file doesn't exist
		confFile = filepath.Join(dir, "configurations", "config_default")
	}

	gcloudCfg, err := ParseGCloudConfig(confFile)
	if err == nil && gcloudCfg != nil {
		env.Project = gcloudCfg.Project
		env.Account = gcloudCfg.Account
		env.ImpersonateSA = gcloudCfg.ImpersonateSA
	}

	// 3. Parse ADC
	adcPath := filepath.Join(dir, "application_default_credentials.json")
	if _, err := os.Stat(adcPath); err == nil {
		env.ADCExists = true
		adcInfo, err := ParseADC(adcPath)
		if err == nil {
			env.ADC = adcInfo
		}
	}

	return env, nil
}

// GetCurrentState inspects the current shell environment variables and configuration files.
func GetCurrentState() (*CurrentState, error) {
	state := &CurrentState{
		CloudSDKConfig:       os.Getenv("CLOUDSDK_CONFIG"),
		GCPEnv:               os.Getenv("GCP_ENV"),
		Project:              os.Getenv("GOOGLE_PROJECT"),
		QuotaProject:         os.Getenv("GOOGLE_CLOUD_QUOTA_PROJECT"),
		GoogleAppCredentials: os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
	}

	envsDir := filepath.Clean(GetEnvsDir())

	var configDir string
	if state.CloudSDKConfig == "" {
		configDir = GetGlobalGCloudDir()
		state.IsCustomConfig = false
		state.ActiveEnvName = ""
	} else {
		configDir = filepath.Clean(state.CloudSDKConfig)
		parent := filepath.Dir(configDir)

		if state.GCPEnv != "" {
			state.ActiveEnvName = state.GCPEnv
		} else if parent == envsDir {
			state.ActiveEnvName = filepath.Base(configDir)
			state.DerivedFromPath = true
		} else {
			state.IsCustomConfig = true
		}
	}

	// Read active configuration name
	activeCfgName := "default"
	activeCfgBytes, err := os.ReadFile(filepath.Join(configDir, "active_config"))
	if err == nil {
		trimmed := strings.TrimSpace(string(activeCfgBytes))
		if trimmed != "" {
			activeCfgName = trimmed
		}
	}
	state.ActiveConfig = activeCfgName

	// Parse INI config
	confFile := filepath.Join(configDir, "configurations", "config_"+activeCfgName)
	if _, err := os.Stat(confFile); os.IsNotExist(err) {
		confFile = filepath.Join(configDir, "configurations", "config_default")
	}

	gcloudCfg, err := ParseGCloudConfig(confFile)
	if err == nil && gcloudCfg != nil {
		if state.Project == "" {
			state.Project = gcloudCfg.Project
		}
		state.Account = gcloudCfg.Account
		state.ImpersonateSA = gcloudCfg.ImpersonateSA
	}

	// Parse ADC
	adcPath := state.GoogleAppCredentials
	if adcPath == "" {
		adcPath = filepath.Join(configDir, "application_default_credentials.json")
	}

	if _, err := os.Stat(adcPath); err == nil {
		adcInfo, err := ParseADC(adcPath)
		if err == nil {
			state.ADC = adcInfo
			if state.QuotaProject == "" && adcInfo.QuotaProjectID != "" {
				state.QuotaProject = adcInfo.QuotaProjectID
			}
		}
	}

	if state.QuotaProject == "" && state.Project != "" {
		state.QuotaProject = state.Project
	}

	return state, nil
}

// GenerateShellExports returns shell commands to set the environment for zsh / bash.
func GenerateShellExports(envName string) (string, error) {
	env, err := GetEnvironment(envName)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("# gcx environment switch: " + envName + "\n")
	sb.WriteString("unset CLOUDSDK_CORE_PROJECT GOOGLE_PROJECT GOOGLE_CLOUD_PROJECT\n")
	sb.WriteString("unset GOOGLE_CLOUD_QUOTA_PROJECT\n")
	sb.WriteString("unset GOOGLE_APPLICATION_CREDENTIALS GOOGLE_IMPERSONATE_SERVICE_ACCOUNT\n")
	sb.WriteString("unset CLOUDSDK_AUTH_IMPERSONATE_SERVICE_ACCOUNT\n")
	sb.WriteString(fmt.Sprintf("export CLOUDSDK_CONFIG=%q\n", env.Path))
	sb.WriteString(fmt.Sprintf("export GCP_ENV=%q\n", env.Name))

	adcPath := filepath.Join(env.Path, "application_default_credentials.json")
	if _, err := os.Stat(adcPath); err == nil {
		sb.WriteString(fmt.Sprintf("export GOOGLE_APPLICATION_CREDENTIALS=%q\n", adcPath))
	}

	if env.Project != "" {
		sb.WriteString(fmt.Sprintf("export GOOGLE_PROJECT=%q\n", env.Project))
		sb.WriteString(fmt.Sprintf("export CLOUDSDK_CORE_PROJECT=%q\n", env.Project))
		sb.WriteString(fmt.Sprintf("export GOOGLE_CLOUD_QUOTA_PROJECT=%q\n", env.Project))
	}

	return sb.String(), nil
}

// GenerateShellUnset returns shell commands to unset all gcx environment variables.
func GenerateShellUnset() string {
	var sb strings.Builder
	sb.WriteString("# gcx environment unset (revert to global gcloud)\n")
	sb.WriteString("unset CLOUDSDK_CONFIG GCP_ENV\n")
	sb.WriteString("unset CLOUDSDK_CORE_PROJECT GOOGLE_PROJECT GOOGLE_CLOUD_PROJECT\n")
	sb.WriteString("unset GOOGLE_CLOUD_QUOTA_PROJECT\n")
	sb.WriteString("unset GOOGLE_APPLICATION_CREDENTIALS GOOGLE_IMPERSONATE_SERVICE_ACCOUNT\n")
	sb.WriteString("unset CLOUDSDK_AUTH_IMPERSONATE_SERVICE_ACCOUNT\n")
	return sb.String()
}
