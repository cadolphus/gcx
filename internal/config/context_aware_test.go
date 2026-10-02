package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cadolphus/gcx/internal/config"
)

// writeGlobalTree creates a minimal global gcloud tree with the given active config body.
func writeGlobalTree(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "configurations"), 0700); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(dir, "active_config"), []byte("corp\n"), 0600)
	_ = os.WriteFile(filepath.Join(dir, "configurations", "config_corp"), []byte(body), 0600)
	return dir
}

func envFrom(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func existsIn(paths ...string) func(string) bool {
	set := map[string]bool{}
	for _, p := range paths {
		set[p] = true
	}
	return func(p string) bool { return set[p] }
}

// Scenario 1/2: cloudtop login shell - puppet env vars present, /etc/gcloud cert config exists.
func TestDetectHostContextAware_CloudtopLoginShell(t *testing.T) {
	global := writeGlobalTree(t, "[core]\naccount = me@google.com\n")
	sys := "/etc/gcloud/certificate_config.json"
	got := config.DetectHostContextAwareWith(config.HostProbe{
		Getenv: envFrom(map[string]string{
			config.EnvContextAwareUseClientCertificate:  "true",
			config.EnvContextAwareCertificateConfigPath: sys,
			config.EnvContextAwareUseECPHTTPProxy:       "true",
			config.EnvContextAwareUseMTLSForGRPC:        "true",
		}),
		GlobalConfigDir:      global,
		SystemCertConfigPath: sys,
		FileExists:           existsIn(sys),
	})
	if got == nil {
		t.Fatal("expected host CAA settings, got nil")
	}
	want := config.ContextAwareSettings{
		UseClientCertificate: "true", CertificateConfigPath: sys, UseECPHTTPProxy: "true", UseMTLSForGRPC: "true",
	}
	if got.Settings != want {
		t.Errorf("settings mismatch:\n got %+v\nwant %+v", got.Settings, want)
	}
	if got.Source != "environment" {
		t.Errorf("expected source environment, got %q", got.Source)
	}
}

// Scenario 1 (the reported bug): cloudtop non-login shell - no env vars, but the global config
// carries use_client_certificate and the system cert config exists.
func TestDetectHostContextAware_CloudtopNonLoginShell(t *testing.T) {
	global := writeGlobalTree(t, "[core]\naccount = me@google.com\n\n[context_aware]\nuse_client_certificate = true\n")
	sys := "/etc/gcloud/certificate_config.json"
	got := config.DetectHostContextAwareWith(config.HostProbe{
		Getenv:               envFrom(nil),
		GlobalConfigDir:      global,
		SystemCertConfigPath: sys,
		FileExists:           existsIn(sys),
	})
	if got == nil {
		t.Fatal("expected host CAA settings, got nil")
	}
	if got.Settings.UseClientCertificate != "true" {
		t.Errorf("expected use_client_certificate=true, got %q", got.Settings.UseClientCertificate)
	}
	if got.Settings.CertificateConfigPath != sys {
		t.Errorf("expected system cert path to be discovered, got %q", got.Settings.CertificateConfigPath)
	}
	if got.Source != "global gcloud config" {
		t.Errorf("expected source 'global gcloud config', got %q", got.Source)
	}
}

// Scenario 3/4: gMac - no env vars, no /etc/gcloud, cert config lives in ~/.config/gcloud.
func TestDetectHostContextAware_GMac(t *testing.T) {
	global := writeGlobalTree(t, "[core]\naccount = me@google.com\n")
	macCert := filepath.Join(global, config.CertificateConfigFileName)
	got := config.DetectHostContextAwareWith(config.HostProbe{
		Getenv:               envFrom(nil),
		GlobalConfigDir:      global,
		SystemCertConfigPath: "/etc/gcloud/certificate_config.json",
		FileExists:           existsIn(macCert),
	})
	if got == nil {
		t.Fatal("expected host CAA settings on gMac, got nil")
	}
	if got.Settings.CertificateConfigPath != macCert {
		t.Errorf("expected %s, got %q", macCert, got.Settings.CertificateConfigPath)
	}
	if got.Settings.UseClientCertificate != "true" {
		t.Errorf("expected cert file presence to imply use_client_certificate=true, got %q", got.Settings.UseClientCertificate)
	}
	if got.Settings.UseECPHTTPProxy != "" || got.Settings.UseMTLSForGRPC != "" {
		t.Errorf("must not invent proxy/grpc settings the host never set: %+v", got.Settings)
	}
}

// Scenario 5: non-Google device - no signal of any kind. Must return nil so nothing is written.
func TestDetectHostContextAware_UnmanagedDevice(t *testing.T) {
	global := writeGlobalTree(t, "[core]\naccount = me@example.com\nproject = p\n")
	got := config.DetectHostContextAwareWith(config.HostProbe{
		Getenv:               envFrom(nil),
		GlobalConfigDir:      global,
		SystemCertConfigPath: "/etc/gcloud/certificate_config.json",
		FileExists:           existsIn(),
	})
	if got != nil {
		t.Fatalf("expected nil on unmanaged device, got %+v", got.Settings)
	}
}

// Explicit opt-out: host says use_client_certificate=false. Propagate nothing.
func TestDetectHostContextAware_ExplicitFalse(t *testing.T) {
	global := writeGlobalTree(t, "[context_aware]\nuse_client_certificate = false\n")
	sys := "/etc/gcloud/certificate_config.json"
	got := config.DetectHostContextAwareWith(config.HostProbe{
		Getenv:               envFrom(nil),
		GlobalConfigDir:      global,
		SystemCertConfigPath: sys,
		FileExists:           existsIn(sys),
	})
	if got != nil {
		t.Fatalf("expected nil when host explicitly disables client cert, got %+v", got.Settings)
	}
}

// Env var points at a cert config that no longer exists (measured: gcloud hard-fails on this).
// The path must be dropped and reported, but the flag still inherited.
func TestDetectHostContextAware_NeverPersistsDeadCertPath(t *testing.T) {
	global := writeGlobalTree(t, "")
	got := config.DetectHostContextAwareWith(config.HostProbe{
		Getenv: envFrom(map[string]string{
			config.EnvContextAwareUseClientCertificate:  "true",
			config.EnvContextAwareCertificateConfigPath: "/nonexistent/certificate_config.json",
		}),
		GlobalConfigDir: global,
		FileExists:      existsIn(),
	})
	if got == nil {
		t.Fatal("expected settings (flag only), got nil")
	}
	if got.Settings.CertificateConfigPath != "" {
		t.Errorf("dead cert path must not be propagated, got %q", got.Settings.CertificateConfigPath)
	}
	if got.DroppedCertPath != "/nonexistent/certificate_config.json" {
		t.Errorf("expected dropped path to be reported, got %q", got.DroppedCertPath)
	}
	if got.Settings.UseClientCertificate != "true" {
		t.Errorf("flag should still be inherited, got %q", got.Settings.UseClientCertificate)
	}
}

// Env takes precedence over the global config file, field by field.
func TestDetectHostContextAware_EnvOverridesGlobalConfig(t *testing.T) {
	global := writeGlobalTree(t, "[context_aware]\nuse_client_certificate = true\nuse_mtls_for_grpc = false\n")
	got := config.DetectHostContextAwareWith(config.HostProbe{
		Getenv: envFrom(map[string]string{
			config.EnvContextAwareUseMTLSForGRPC: "true",
		}),
		GlobalConfigDir: global,
		FileExists:      existsIn(),
	})
	if got == nil {
		t.Fatal("expected settings, got nil")
	}
	if got.Settings.UseMTLSForGRPC != "true" {
		t.Errorf("env should win over config, got use_mtls_for_grpc=%q", got.Settings.UseMTLSForGRPC)
	}
	if got.Settings.UseClientCertificate != "true" {
		t.Errorf("unset env field should fall back to config, got %q", got.Settings.UseClientCertificate)
	}
}

func TestContextAwareSettings_Properties(t *testing.T) {
	s := config.ContextAwareSettings{UseClientCertificate: "true", CertificateConfigPath: "/x.json"}
	props := s.Properties()
	if len(props) != 2 {
		t.Fatalf("expected 2 properties, got %d: %v", len(props), props)
	}
	if props[0] != [2]string{"context_aware/use_client_certificate", "true"} {
		t.Errorf("unexpected first property %v", props[0])
	}
	if props[1] != [2]string{"context_aware/certificate_config_file_path", "/x.json"} {
		t.Errorf("unexpected second property %v", props[1])
	}
}

func TestDiagnoseTreeContextAware(t *testing.T) {
	host := &config.HostContextAware{Settings: config.ContextAwareSettings{
		UseClientCertificate: "true", CertificateConfigPath: "/etc/gcloud/certificate_config.json",
	}}

	t.Run("bare tree on managed host -> two non-fatal issues", func(t *testing.T) {
		issues := config.DiagnoseTreeContextAware(host, config.ContextAwareSettings{}, existsIn())
		if len(issues) != 2 {
			t.Fatalf("expected 2 issues, got %d: %+v", len(issues), issues)
		}
		for _, i := range issues {
			if i.Fatal {
				t.Errorf("expected non-fatal, got fatal: %s", i.Message)
			}
		}
	})

	t.Run("tree matches host -> clean", func(t *testing.T) {
		issues := config.DiagnoseTreeContextAware(host, host.Settings, existsIn(host.Settings.CertificateConfigPath))
		if len(issues) != 0 {
			t.Errorf("expected no issues, got %+v", issues)
		}
	})

	t.Run("stale cert path is fatal even on unmanaged host", func(t *testing.T) {
		issues := config.DiagnoseTreeContextAware(nil, config.ContextAwareSettings{
			UseClientCertificate: "true", CertificateConfigPath: "/gone.json",
		}, existsIn())
		if len(issues) != 1 || !issues[0].Fatal {
			t.Fatalf("expected one fatal issue, got %+v", issues)
		}
	})

	t.Run("unmanaged host, bare tree -> clean", func(t *testing.T) {
		issues := config.DiagnoseTreeContextAware(nil, config.ContextAwareSettings{}, existsIn())
		if len(issues) != 0 {
			t.Errorf("expected no issues, got %+v", issues)
		}
	})
}

func TestParseGCloudConfig_ContextAwareSection(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config_x")
	_ = os.WriteFile(p, []byte("[core]\nproject = p\n\n[context_aware]\nuse_client_certificate = true\ncertificate_config_file_path = /etc/gcloud/certificate_config.json\n"), 0600)
	cfg, err := config.ParseGCloudConfig(p)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ContextAware.UseClientCertificate != "true" || cfg.ContextAware.CertificateConfigPath != "/etc/gcloud/certificate_config.json" {
		t.Errorf("unexpected context_aware parse: %+v", cfg.ContextAware)
	}
}
