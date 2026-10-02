package config

import (
	"fmt"
	"os"

	"gopkg.in/ini.v1"
)

// GCloudConfig holds parsed settings from a gcloud configurations/config_<name> file.
type GCloudConfig struct {
	Project       string
	Account       string
	ImpersonateSA string
	ComputeRegion string
	ComputeZone   string
	// ContextAware holds the raw [context_aware] section values ("" when unset).
	ContextAware ContextAwareSettings
}

// ParseGCloudConfig parses a gcloud config file at the given path.
func ParseGCloudConfig(filePath string) (*GCloudConfig, error) {
	cfg := &GCloudConfig{}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return cfg, nil
	}

	iniFile, err := ini.Load(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load gcloud config file %s: %w", filePath, err)
	}

	coreSec := iniFile.Section("core")
	if coreSec != nil {
		cfg.Project = coreSec.Key("project").String()
		cfg.Account = coreSec.Key("account").String()
	}

	authSec := iniFile.Section("auth")
	if authSec != nil {
		cfg.ImpersonateSA = authSec.Key("impersonate_service_account").String()
	}

	computeSec := iniFile.Section("compute")
	if computeSec != nil {
		cfg.ComputeRegion = computeSec.Key("region").String()
		cfg.ComputeZone = computeSec.Key("zone").String()
	}

	caSec := iniFile.Section("context_aware")
	if caSec != nil {
		cfg.ContextAware = ContextAwareSettings{
			UseClientCertificate:  caSec.Key(ctxAwareKeyUseClientCertificate).String(),
			CertificateConfigPath: caSec.Key(ctxAwareKeyCertificateConfigPath).String(),
			UseECPHTTPProxy:       caSec.Key(ctxAwareKeyUseECPHTTPProxy).String(),
			UseMTLSForGRPC:        caSec.Key(ctxAwareKeyUseMTLSForGRPC).String(),
		}
	}

	return cfg, nil
}
