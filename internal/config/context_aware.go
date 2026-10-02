package config

import (
	"os"
	"path/filepath"
	"strings"
)

// gcloud [context_aware] property names (as used by `gcloud config set context_aware/<key>`).
const (
	ctxAwareKeyUseClientCertificate  = "use_client_certificate"
	ctxAwareKeyCertificateConfigPath = "certificate_config_file_path"
	ctxAwareKeyUseECPHTTPProxy       = "use_ecp_http_proxy"
	ctxAwareKeyUseMTLSForGRPC        = "use_mtls_for_grpc"
)

// Environment variables gcloud honours in preference to the property file.
// On managed gLinux/cloudtop hosts these are exported by /etc/profile.d/ecp_gcloud_env.sh,
// which only runs for login shells - the root cause of intermittent CAA failures.
const (
	EnvContextAwareUseClientCertificate  = "CLOUDSDK_CONTEXT_AWARE_USE_CLIENT_CERTIFICATE"
	EnvContextAwareCertificateConfigPath = "CLOUDSDK_CONTEXT_AWARE_CERTIFICATE_CONFIG_FILE_PATH"
	EnvContextAwareUseECPHTTPProxy       = "CLOUDSDK_CONTEXT_AWARE_USE_ECP_HTTP_PROXY"
	EnvContextAwareUseMTLSForGRPC        = "CLOUDSDK_CONTEXT_AWARE_USE_MTLS_FOR_GRPC"
	// EnvGoogleAPICertificateConfig is the client-library equivalent of the cert path; used as a fallback source.
	EnvGoogleAPICertificateConfig = "GOOGLE_API_CERTIFICATE_CONFIG"
)

// CertificateConfigFileName is the file gcloud looks for inside CLOUDSDK_CONFIG when no path property is set.
const CertificateConfigFileName = "certificate_config.json"

// systemCertificateConfigPath is where puppet installs the ECP config on managed gLinux hosts.
const systemCertificateConfigPath = "/etc/gcloud/certificate_config.json"

// ContextAwareSettings is the subset of gcloud [context_aware] properties that gcx carries
// into isolated trees. Values are raw gcloud property strings; "" means unset.
type ContextAwareSettings struct {
	UseClientCertificate  string `json:"use_client_certificate,omitempty"`
	CertificateConfigPath string `json:"certificate_config_file_path,omitempty"`
	UseECPHTTPProxy       string `json:"use_ecp_http_proxy,omitempty"`
	UseMTLSForGRPC        string `json:"use_mtls_for_grpc,omitempty"`
}

// IsEmpty reports whether no [context_aware] property is set.
func (s ContextAwareSettings) IsEmpty() bool {
	return s.UseClientCertificate == "" && s.CertificateConfigPath == "" &&
		s.UseECPHTTPProxy == "" && s.UseMTLSForGRPC == ""
}

// ClientCertificateEnabled reports whether use_client_certificate is truthy per gcloud's boolean parsing.
func (s ContextAwareSettings) ClientCertificateEnabled() bool {
	return isGCloudTrue(s.UseClientCertificate)
}

// Properties returns the settings as ordered (key, value) pairs suitable for
// `gcloud config set context_aware/<key> <value>`. Unset values are omitted.
func (s ContextAwareSettings) Properties() [][2]string {
	var out [][2]string
	add := func(k, v string) {
		if v != "" {
			out = append(out, [2]string{"context_aware/" + k, v})
		}
	}
	add(ctxAwareKeyUseClientCertificate, s.UseClientCertificate)
	add(ctxAwareKeyCertificateConfigPath, s.CertificateConfigPath)
	add(ctxAwareKeyUseECPHTTPProxy, s.UseECPHTTPProxy)
	add(ctxAwareKeyUseMTLSForGRPC, s.UseMTLSForGRPC)
	return out
}

// HostContextAware is the result of probing the host for Context Aware Access configuration.
type HostContextAware struct {
	Settings ContextAwareSettings
	// Source describes where the decisive signal came from (for display).
	Source string
	// DroppedCertPath is a cert path that was advertised by env/config but does not exist on disk.
	// It is never propagated, because gcloud hard-fails on a non-existent certificate_config_file_path.
	DroppedCertPath string
}

// HostProbe abstracts the host so detection is unit-testable per device scenario.
type HostProbe struct {
	Getenv          func(string) string
	GlobalConfigDir string
	// SystemCertConfigPath is the OS-level cert config location (e.g. /etc/gcloud/certificate_config.json). May be "".
	SystemCertConfigPath string
	FileExists           func(string) bool
}

// DefaultHostProbe returns a probe backed by the real process environment and filesystem.
func DefaultHostProbe() HostProbe {
	return HostProbe{
		Getenv:               os.Getenv,
		GlobalConfigDir:      GetGlobalGCloudDir(),
		SystemCertConfigPath: systemCertificateConfigPath,
		FileExists:           fileExists,
	}
}

// DetectHostContextAware probes the real host. See DetectHostContextAwareWith.
func DetectHostContextAware() *HostContextAware {
	return DetectHostContextAwareWith(DefaultHostProbe())
}

// DetectHostContextAwareWith returns the [context_aware] properties the host already uses for
// the global gcloud tree, so they can be replicated into an isolated tree. Precedence:
//
//  1. CLOUDSDK_CONTEXT_AWARE_* environment variables (what gcloud itself prefers)
//  2. [context_aware] section of the global active configuration
//  3. Presence of a certificate_config.json at <global>/ or the system path
//
// A certificate_config_file_path is only returned if the file exists at probe time.
// Returns nil when the host exhibits no CAA signal (an unmanaged device), so callers write nothing.
func DetectHostContextAwareWith(p HostProbe) *HostContextAware {
	if p.Getenv == nil {
		p.Getenv = func(string) string { return "" }
	}
	if p.FileExists == nil {
		p.FileExists = func(string) bool { return false }
	}

	var s ContextAwareSettings
	source := ""

	// 1. Environment
	envSettings := ContextAwareSettings{
		UseClientCertificate:  p.Getenv(EnvContextAwareUseClientCertificate),
		CertificateConfigPath: firstNonEmpty(p.Getenv(EnvContextAwareCertificateConfigPath), p.Getenv(EnvGoogleAPICertificateConfig)),
		UseECPHTTPProxy:       p.Getenv(EnvContextAwareUseECPHTTPProxy),
		UseMTLSForGRPC:        p.Getenv(EnvContextAwareUseMTLSForGRPC),
	}
	if !envSettings.IsEmpty() {
		s = envSettings
		source = "environment"
	}

	// 2. Global active configuration (fills only what env left unset)
	if p.GlobalConfigDir != "" {
		if global := readTreeContextAware(p.GlobalConfigDir); global != nil && !global.IsEmpty() {
			merged := mergeContextAware(s, *global)
			if merged != s && source == "" {
				source = "global gcloud config"
			}
			s = merged
		}
	}

	// 3. Well-known certificate config locations
	if s.CertificateConfigPath == "" {
		candidates := []string{}
		if p.GlobalConfigDir != "" {
			candidates = append(candidates, filepath.Join(p.GlobalConfigDir, CertificateConfigFileName))
		}
		if p.SystemCertConfigPath != "" {
			candidates = append(candidates, p.SystemCertConfigPath)
		}
		for _, c := range candidates {
			if p.FileExists(c) {
				s.CertificateConfigPath = c
				if source == "" {
					source = "certificate config file"
				}
				break
			}
		}
	}

	// An explicit "false" means the host has opted out; do not propagate anything.
	if s.UseClientCertificate != "" && !isGCloudTrue(s.UseClientCertificate) {
		return nil
	}

	// A discovered cert config implies the device is set up for client certs.
	if s.CertificateConfigPath != "" && s.UseClientCertificate == "" {
		s.UseClientCertificate = "true"
	}

	result := &HostContextAware{Source: source}

	// Never persist a path gcloud will refuse to load.
	if s.CertificateConfigPath != "" && !p.FileExists(s.CertificateConfigPath) {
		result.DroppedCertPath = s.CertificateConfigPath
		s.CertificateConfigPath = ""
	}

	if !s.ClientCertificateEnabled() {
		return nil
	}

	result.Settings = s
	return result
}

// ReadTreeContextAware returns the [context_aware] section of the active configuration in a tree,
// or nil if the tree has no readable configuration.
func ReadTreeContextAware(configDir string) *ContextAwareSettings {
	return readTreeContextAware(configDir)
}

func readTreeContextAware(configDir string) *ContextAwareSettings {
	activeCfgName := "default"
	if b, err := os.ReadFile(filepath.Join(configDir, "active_config")); err == nil {
		if t := strings.TrimSpace(string(b)); t != "" {
			activeCfgName = t
		}
	}
	confFile := filepath.Join(configDir, "configurations", "config_"+activeCfgName)
	if _, err := os.Stat(confFile); os.IsNotExist(err) {
		confFile = filepath.Join(configDir, "configurations", "config_default")
	}
	cfg, err := ParseGCloudConfig(confFile)
	if err != nil || cfg == nil {
		return nil
	}
	s := cfg.ContextAware
	return &s
}

// ContextAwareIssue is a drift finding between what the host needs and what a tree has.
type ContextAwareIssue struct {
	// Fatal issues make gcloud fail outright; non-fatal ones cause CAA denial only in shells
	// lacking the CLOUDSDK_CONTEXT_AWARE_* environment variables.
	Fatal   bool
	Message string
}

// DiagnoseTreeContextAware compares a tree's [context_aware] against the host's requirements.
// host may be nil (unmanaged device), in which case only fatal tree-local problems are reported.
func DiagnoseTreeContextAware(host *HostContextAware, tree ContextAwareSettings, fileExists func(string) bool) []ContextAwareIssue {
	if fileExists == nil {
		fileExists = func(string) bool { return false }
	}
	var issues []ContextAwareIssue

	if tree.CertificateConfigPath != "" && !fileExists(tree.CertificateConfigPath) {
		issues = append(issues, ContextAwareIssue{
			Fatal:   true,
			Message: "certificate_config_file_path points to a missing file: " + tree.CertificateConfigPath + " (gcloud will refuse to run)",
		})
	}

	if host == nil {
		return issues
	}

	if !tree.ClientCertificateEnabled() {
		issues = append(issues, ContextAwareIssue{
			Message: "use_client_certificate not set in tree; gcloud will be blocked by Context Aware Access in shells without CLOUDSDK_CONTEXT_AWARE_* env vars",
		})
	}
	if host.Settings.CertificateConfigPath != "" && tree.CertificateConfigPath == "" {
		issues = append(issues, ContextAwareIssue{
			Message: "certificate_config_file_path not set in tree; host uses " + host.Settings.CertificateConfigPath,
		})
	}
	return issues
}

// mergeContextAware returns base with any unset fields filled from fallback.
func mergeContextAware(base, fallback ContextAwareSettings) ContextAwareSettings {
	out := base
	if out.UseClientCertificate == "" {
		out.UseClientCertificate = fallback.UseClientCertificate
	}
	if out.CertificateConfigPath == "" {
		out.CertificateConfigPath = fallback.CertificateConfigPath
	}
	if out.UseECPHTTPProxy == "" {
		out.UseECPHTTPProxy = fallback.UseECPHTTPProxy
	}
	if out.UseMTLSForGRPC == "" {
		out.UseMTLSForGRPC = fallback.UseMTLSForGRPC
	}
	return out
}

// isGCloudTrue mirrors gcloud's permissive boolean parsing for property values.
func isGCloudTrue(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "on", "yes", "y":
		return true
	}
	return false
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
