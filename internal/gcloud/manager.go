package gcloud

import (
	"fmt"
	"strings"
	"time"
)

// SetupConfigDir initializes and activates a named gcloud configuration in the isolated tree.
func SetupConfigDir(envName, configDir string) error {
	envMap := map[string]string{
		"CLOUDSDK_CONFIG": configDir,
		"GCP_ENV":         envName,
	}

	// Check if named config already exists
	_, err := RunCapture(envMap, "config", "configurations", "describe", envName)
	if err == nil {
		// Already exists - activate it
		_, err = RunCapture(envMap, "config", "configurations", "activate", envName)
		return err
	}

	// Create new configuration
	_, err = RunCapture(envMap, "config", "configurations", "create", envName)
	return err
}

// UnsetImpersonation removes any active CLI impersonation from the configuration.
func UnsetImpersonation(configDir string) error {
	envMap := map[string]string{"CLOUDSDK_CONFIG": configDir}
	_, _ = RunCapture(envMap, "config", "unset", "auth/impersonate_service_account")
	return nil
}

// SetImpersonation sets the CLI impersonation service account.
func SetImpersonation(sa, configDir string) error {
	envMap := map[string]string{"CLOUDSDK_CONFIG": configDir}
	_, err := RunCapture(envMap, "config", "set", "auth/impersonate_service_account", sa)
	return err
}

// SetProject sets the default project in the configuration.
func SetProject(proj, configDir string) error {
	envMap := map[string]string{"CLOUDSDK_CONFIG": configDir}
	_, err := RunCapture(envMap, "config", "set", "project", proj)
	return err
}

// AuthLogin initiates browserless user login for the configuration tree.
func AuthLogin(configDir string) error {
	envMap := map[string]string{"CLOUDSDK_CONFIG": configDir}
	return RunInteractive(envMap, "auth", "login", "--no-launch-browser")
}

// GetAccount retrieves the currently authenticated user account.
func GetAccount(configDir string) (string, error) {
	envMap := map[string]string{"CLOUDSDK_CONFIG": configDir}
	acct, err := RunCapture(envMap, "config", "get-value", "account")
	if err != nil || acct == "" || acct == "(unset)" {
		return "", fmt.Errorf("could not determine authenticated account")
	}
	return acct, nil
}

// GrantUserTokenCreatorRoles grants service account impersonation rights to the user on the project.
func GrantUserTokenCreatorRoles(proj, userAcct, configDir string) error {
	envMap := map[string]string{"CLOUDSDK_CONFIG": configDir}
	roles := []string{
		"roles/iam.serviceAccountTokenCreator",
		"roles/iam.serviceAccountUser",
	}

	for _, role := range roles {
		_, err := RunCapture(envMap, "projects", "add-iam-policy-binding", proj,
			fmt.Sprintf("--member=user:%s", userAcct),
			fmt.Sprintf("--role=%s", role),
			"--condition=None",
		)
		if err != nil {
			return fmt.Errorf("failed to grant %s to user:%s on project %s: %w", role, userAcct, proj, err)
		}
	}
	return nil
}

// ServiceAccountExists checks if a service account exists in the project.
func ServiceAccountExists(proj, sa, configDir string) (bool, error) {
	envMap := map[string]string{"CLOUDSDK_CONFIG": configDir}
	_, err := RunCapture(envMap, "iam", "service-accounts", "describe", sa, "--project="+proj)
	if err == nil {
		return true, nil
	}
	return false, nil
}

// CreateServiceAccount creates a new service account in the project.
func CreateServiceAccount(proj, saName, configDir string) error {
	envMap := map[string]string{"CLOUDSDK_CONFIG": configDir}
	_, err := RunCapture(envMap, "iam", "service-accounts", "create", saName,
		"--project="+proj,
		"--display-name="+saName,
	)
	if err != nil {
		return fmt.Errorf("failed to create service account '%s' in project '%s': %w", saName, proj, err)
	}
	return nil
}

// HasRoleBinding checks if the service account already has the given project role.
func HasRoleBinding(proj, role, member, configDir string) (bool, error) {
	envMap := map[string]string{"CLOUDSDK_CONFIG": configDir}
	filter := fmt.Sprintf("bindings.role=%s AND bindings.members=%s", role, member)
	out, err := RunCapture(envMap, "projects", "get-iam-policy", proj,
		"--flatten=bindings[].members",
		"--filter="+filter,
		"--format=value(bindings.members)",
	)
	if err == nil && strings.TrimSpace(out) != "" {
		return true, nil
	}
	return false, nil
}

// GrantSARolesWithRetry grants required roles to the service account with retry backoff for IAM eventual consistency.
func GrantSARolesWithRetry(proj, saEmail, configDir string, progressFn func(msg string)) error {
	envMap := map[string]string{"CLOUDSDK_CONFIG": configDir}
	member := "serviceAccount:" + saEmail
	roles := []string{
		"roles/editor",
		"roles/resourcemanager.projectIamAdmin",
	}

	for _, role := range roles {
		has, _ := HasRoleBinding(proj, role, member, configDir)
		if has {
			if progressFn != nil {
				progressFn(fmt.Sprintf("Already has %s", role))
			}
			continue
		}

		if progressFn != nil {
			progressFn(fmt.Sprintf("Granting %s to %s on %s...", role, member, proj))
		}

		attempt := 1
		maxAttempts := 5
		for {
			stdout, stderr, err := RunCaptureAll(envMap, "projects", "add-iam-policy-binding", proj,
				"--member="+member,
				"--role="+role,
				"--condition=None",
			)
			if err == nil {
				break
			}

			combined := stderr + " " + stdout
			if !strings.Contains(combined, "does not exist") || attempt >= maxAttempts {
				return fmt.Errorf("failed to grant %s to %s: %s (%w)", role, member, combined, err)
			}

			waitSec := attempt * 2
			if progressFn != nil {
				progressFn(fmt.Sprintf("IAM propagation pending for '%s'; retry %d/%d in %ds...", saEmail, attempt, maxAttempts, waitSec))
			}
			time.Sleep(time.Duration(waitSec) * time.Second)
			attempt++
		}
	}

	return nil
}

// AuthApplicationDefaultLogin initiates application-default credentials login (with optional impersonation).
func AuthApplicationDefaultLogin(configDir, impersonateSA string) error {
	envMap := map[string]string{"CLOUDSDK_CONFIG": configDir}
	args := []string{"auth", "application-default", "login", "--no-launch-browser"}
	if impersonateSA != "" {
		args = append(args, "--impersonate-service-account="+impersonateSA)
	}
	return RunInteractive(envMap, args...)
}
