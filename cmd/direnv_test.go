package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDirenvCommandExportsGoogleCloudProject(t *testing.T) {
	tempHome := t.TempDir()
	envsDir := filepath.Join(tempHome, ".gcloud-envs")
	t.Setenv("GCX_HOME", envsDir)

	withProject := filepath.Join(envsDir, "with-proj")
	if err := os.MkdirAll(filepath.Join(withProject, "configurations"), 0700); err != nil {
		t.Fatalf("failed to create env dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(withProject, "active_config"), []byte("with-proj\n"), 0600); err != nil {
		t.Fatalf("failed to write active_config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(withProject, "configurations", "config_with-proj"), []byte("[core]\nproject = target-proj-123\n"), 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	out := runDirenvForTest(t, "with-proj")
	for _, expected := range []string{
		`export GOOGLE_PROJECT="target-proj-123"`,
		`export GOOGLE_CLOUD_PROJECT="target-proj-123"`,
		`export CLOUDSDK_CORE_PROJECT="target-proj-123"`,
		`export GOOGLE_CLOUD_QUOTA_PROJECT="target-proj-123"`,
	} {
		if !strings.Contains(out, expected) {
			t.Errorf("expected direnv output to contain %q, got:\n%s", expected, out)
		}
	}

	withoutProject := filepath.Join(envsDir, "no-proj")
	if err := os.MkdirAll(filepath.Join(withoutProject, "configurations"), 0700); err != nil {
		t.Fatalf("failed to create env dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(withoutProject, "active_config"), []byte("no-proj\n"), 0600); err != nil {
		t.Fatalf("failed to write active_config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(withoutProject, "configurations", "config_no-proj"), []byte("[core]\naccount = user@example.com\n"), 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	outNoProj := runDirenvForTest(t, "no-proj")
	expectedUnset := "unset GOOGLE_PROJECT GOOGLE_CLOUD_PROJECT CLOUDSDK_CORE_PROJECT GOOGLE_CLOUD_QUOTA_PROJECT"
	if !strings.Contains(outNoProj, expectedUnset) {
		t.Errorf("expected direnv output without project to contain %q, got:\n%s", expectedUnset, outNoProj)
	}
}

func runDirenvForTest(t *testing.T, envName string) string {
	t.Helper()
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	runErr := direnvCmd.RunE(direnvCmd, []string{envName})
	_ = w.Close()
	os.Stdout = origStdout

	if runErr != nil {
		t.Fatalf("direnvCmd.RunE(%q) failed: %v", envName, runErr)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read captured stdout: %v", err)
	}
	return buf.String()
}
