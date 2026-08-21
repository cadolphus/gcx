package gcloud

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CheckInstalled verifies that gcloud CLI is available in PATH.
func CheckInstalled() error {
	_, err := exec.LookPath("gcloud")
	if err != nil {
		return fmt.Errorf("gcloud CLI not found in PATH. Please install Google Cloud SDK: https://cloud.google.com/sdk/docs/install")
	}
	return nil
}

// CleanEnv returns an environment slice stripped of overriding sticky variables.
func CleanEnv(extra map[string]string) []string {
	var envs []string
	// Keys to remove to prevent sticky overrides
	blocked := map[string]bool{
		"CLOUDSDK_CORE_PROJECT":                     true,
		"GOOGLE_PROJECT":                            true,
		"GOOGLE_CLOUD_PROJECT":                      true,
		"GOOGLE_CLOUD_QUOTA_PROJECT":                true,
		"GOOGLE_APPLICATION_CREDENTIALS":            true,
		"GOOGLE_IMPERSONATE_SERVICE_ACCOUNT":        true,
		"CLOUDSDK_AUTH_IMPERSONATE_SERVICE_ACCOUNT": true,
	}

	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) > 0 && blocked[parts[0]] {
			continue
		}
		envs = append(envs, e)
	}

	for k, v := range extra {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}

	return envs
}

// RunInteractive executes a gcloud command connected directly to os.Stdin/Stdout/Stderr.
func RunInteractive(extraEnv map[string]string, args ...string) error {
	if err := CheckInstalled(); err != nil {
		return err
	}

	cmd := exec.Command("gcloud", args...)
	cmd.Env = CleanEnv(extraEnv)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// RunSilent executes a gcloud command discarding standard outputs.
func RunSilent(extraEnv map[string]string, args ...string) error {
	if err := CheckInstalled(); err != nil {
		return err
	}

	cmd := exec.Command("gcloud", args...)
	cmd.Env = CleanEnv(extraEnv)
	return cmd.Run()
}

// RunCapture executes a gcloud command and returns trimmed stdout.
func RunCapture(extraEnv map[string]string, args ...string) (string, error) {
	if err := CheckInstalled(); err != nil {
		return "", err
	}

	cmd := exec.Command("gcloud", args...)
	cmd.Env = CleanEnv(extraEnv)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = strings.TrimSpace(stdout.String())
		}
		return "", fmt.Errorf("%s (exit %v)", errMsg, err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// RunCaptureAll executes a gcloud command returning stdout, stderr, and error.
func RunCaptureAll(extraEnv map[string]string, args ...string) (string, string, error) {
	if err := CheckInstalled(); err != nil {
		return "", "", err
	}

	cmd := exec.Command("gcloud", args...)
	cmd.Env = CleanEnv(extraEnv)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
}
