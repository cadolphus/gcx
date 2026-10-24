package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cadolphus/gcx/internal/config"
)

func TestParseADC(t *testing.T) {
	tempDir := t.TempDir()
	adcPath := filepath.Join(tempDir, "application_default_credentials.json")

	content := `{
  "type": "impersonated_service_account",
  "service_account_impersonation_url": "https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/my-sa@my-proj.iam.gserviceaccount.com:generateAccessToken",
  "source_credentials": {
    "type": "authorized_user"
  },
  "quota_project_id": "my-quota-proj"
}`

	if err := os.WriteFile(adcPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test ADC file: %v", err)
	}

	info, err := config.ParseADC(adcPath)
	if err != nil {
		t.Fatalf("unexpected error parsing ADC: %v", err)
	}

	if info.Type != "impersonated_service_account" {
		t.Errorf("expected type impersonated_service_account, got %s", info.Type)
	}
	if info.ImpersonatedEmail != "my-sa@my-proj.iam.gserviceaccount.com" {
		t.Errorf("expected impersonated email my-sa@my-proj.iam.gserviceaccount.com, got %s", info.ImpersonatedEmail)
	}
	if info.QuotaProjectID != "my-quota-proj" {
		t.Errorf("expected quota project my-quota-proj, got %s", info.QuotaProjectID)
	}
}

func TestParseGCloudConfig(t *testing.T) {
	tempDir := t.TempDir()
	confPath := filepath.Join(tempDir, "config_test")

	content := `[core]
account = test-user@example.com
project = test-project-123

[auth]
impersonate_service_account = deployer@test-project-123.iam.gserviceaccount.com
`

	if err := os.WriteFile(confPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test config file: %v", err)
	}

	cfg, err := config.ParseGCloudConfig(confPath)
	if err != nil {
		t.Fatalf("unexpected error parsing config: %v", err)
	}

	if cfg.Project != "test-project-123" {
		t.Errorf("expected project test-project-123, got %s", cfg.Project)
	}
	if cfg.Account != "test-user@example.com" {
		t.Errorf("expected account test-user@example.com, got %s", cfg.Account)
	}
	if cfg.ImpersonateSA != "deployer@test-project-123.iam.gserviceaccount.com" {
		t.Errorf("expected impersonate SA deployer@test-project-123.iam.gserviceaccount.com, got %s", cfg.ImpersonateSA)
	}
}

func TestGenerateShellExports(t *testing.T) {
	tempHome := t.TempDir()
	envsDir := filepath.Join(tempHome, ".gcloud-envs")
	envDir := filepath.Join(envsDir, "testenv")
	if err := os.MkdirAll(filepath.Join(envDir, "configurations"), 0700); err != nil {
		t.Fatalf("failed to create test env dir: %v", err)
	}

	_ = os.WriteFile(filepath.Join(envDir, "active_config"), []byte("testenv\n"), 0600)
	_ = os.WriteFile(filepath.Join(envDir, "configurations", "config_testenv"), []byte("[core]\nproject = demo-proj\n"), 0600)

	t.Setenv("GCX_HOME", envsDir)

	exports, err := config.GenerateShellExports("testenv")
	if err != nil {
		t.Fatalf("unexpected error generating exports: %v", err)
	}

	if !testing.Short() {
		if len(exports) == 0 {
			t.Errorf("expected non-empty exports")
		}
	}
}

func TestIsNestedEnvironment(t *testing.T) {
	tempDir := t.TempDir()
	nestedEnv := filepath.Join(tempDir, "dev1")
	if err := os.MkdirAll(filepath.Join(nestedEnv, "configurations"), 0700); err != nil {
		t.Fatal(err)
	}

	if !config.IsNestedEnvironment(nestedEnv) {
		t.Errorf("expected %s to be recognized as nested environment", nestedEnv)
	}

	plainDir := filepath.Join(tempDir, "plain")
	if err := os.MkdirAll(plainDir, 0700); err != nil {
		t.Fatal(err)
	}
	if config.IsNestedEnvironment(plainDir) {
		t.Errorf("expected plain directory not to be recognized as nested environment")
	}
}

func TestFlattenNestedEnvironment(t *testing.T) {
	tempHome := t.TempDir()
	envsDir := filepath.Join(tempHome, ".gcloud-envs")
	parentDir := filepath.Join(envsDir, "dev1")
	nestedDir := filepath.Join(parentDir, "dev1")

	// Create parent with empty configurations
	if err := os.MkdirAll(filepath.Join(parentDir, "configurations"), 0700); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(parentDir, "active_config"), []byte("default\n"), 0600)

	// Create nested directory with actual environment data
	if err := os.MkdirAll(filepath.Join(nestedDir, "configurations"), 0700); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(nestedDir, "active_config"), []byte("dev1\n"), 0600)
	_ = os.WriteFile(filepath.Join(nestedDir, "configurations", "config_dev1"), []byte("[core]\nproject = test-proj\naccount = user@test.com\n"), 0600)
	_ = os.WriteFile(filepath.Join(nestedDir, "application_default_credentials.json"), []byte(`{"type": "authorized_user"}`), 0600)

	t.Setenv("GCX_HOME", envsDir)

	if !config.IsNestedEnvironment(nestedDir) {
		t.Fatalf("expected nestedDir to be identified as nested environment")
	}

	if err := config.FlattenNestedEnvironment(parentDir, nestedDir); err != nil {
		t.Fatalf("FlattenNestedEnvironment failed: %v", err)
	}

	if _, err := os.Stat(nestedDir); !os.IsNotExist(err) {
		t.Errorf("expected nestedDir to be removed, but stat succeeded")
	}

	env, err := config.GetEnvironment("dev1")
	if err != nil {
		t.Fatalf("GetEnvironment failed after flatten: %v", err)
	}

	if env.Project != "test-proj" {
		t.Errorf("expected project test-proj, got %s", env.Project)
	}
	if env.Account != "user@test.com" {
		t.Errorf("expected account user@test.com, got %s", env.Account)
	}
	if !env.ADCExists {
		t.Errorf("expected ADC to exist after flatten")
	}
}
