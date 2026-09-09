# gcx

`gcx` is a fast, self-contained CLI tool for managing isolated Google Cloud SDK (`gcloud`) configuration trees, Application Default Credentials (ADC), and Service Account impersonation.

Designed for sub-millisecond environment switching and seamless [`direnv`](https://direnv.net/) integration.

---

## Features

- **Isolated Trees**: Each environment maintains its own `CLOUDSDK_CONFIG`, token cache, and ADC file.
- **Fast Execution**: Written in Go with native config & JSON parsing (< 3ms response time).
- **Direnv Integration**: Instant per-directory GCP context loading via `use gcx <env>`.
- **Bash & Zsh Support**: Shell function wrapper with dynamic tab completion.
- **Safety**: Automatically clears sticky env vars (`CLOUDSDK_CORE_PROJECT`, `GOOGLE_PROJECT`, etc.) preventing credential drift.

---

## Installation

> [!IMPORTANT]
> **Universal Requirement**: The **Google Cloud SDK (`gcloud`)** must be installed and available in your `PATH` for all installation methods.

---

### Option 1: Quick Install (Pre-built Binaries)

**Requirements**: `curl`, `tar`, and write access to `/usr/local/bin` (no Go compiler or build tools needed).

Detects your OS and architecture, resolves the current release, and installs into `/usr/local/bin`:

```bash
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m); [ "$ARCH" = "x86_64" ] && ARCH=amd64; [ "$ARCH" = "aarch64" ] && ARCH=arm64
VER=$(curl -sL https://api.github.com/repos/cadolphus/gcx/releases/latest | sed -n 's/.*"tag_name": *"v\{0,1\}\([^"]*\)".*/\1/p')
curl -sL "https://github.com/cadolphus/gcx/releases/latest/download/gcx_${VER}_${OS}_${ARCH}.tar.gz" \
  | sudo tar -xz -C /usr/local/bin gcx
```

Pre-built binaries are published for `linux_amd64`, `linux_arm64`, `darwin_amd64`, and `darwin_arm64`.

To verify the download before installing:

```bash
curl -sL https://github.com/cadolphus/gcx/releases/latest/download/checksums.txt -o checksums.txt
shasum -a 256 -c checksums.txt --ignore-missing
```

---

### Option 2: With `go install`

**Requirements**: a Go toolchain matching the `go` directive in [`go.mod`](go.mod), and `~/go/bin` in your `PATH` (e.g. `export PATH="$HOME/go/bin:$PATH"`).

```bash
go install github.com/cadolphus/gcx@latest
```

---

### Option 3: Build from Source

**Requirements**: a Go toolchain matching the `go` directive in [`go.mod`](go.mod), plus **make** and **git**.

```bash
git clone https://github.com/cadolphus/gcx.git
cd gcx

# Build and install to ~/go/bin:
make install

# Or build standalone binary into ./bin/gcx and move to /usr/local/bin:
make build
sudo cp bin/gcx /usr/local/bin/
```

---

## Shell Setup

Add **one line** to your shell configuration file:

### Zsh (`~/.zshrc`)
```zsh
eval "$(command gcx init zsh)"
```

### Bash (`~/.bashrc`)
```bash
eval "$(command gcx init bash)"
```

---

## Direnv Integration

Add the `use_gcx` function to your `~/.config/direnv/direnvrc`:

```bash
use_gcx() {
  eval "$(gcx direnv "$1")"
}
```

Then in any project repository's `.envrc`:

```bash
use gcx my-env
```

---

## Commands

| Command | Description |
| :--- | :--- |
| `gcx [env]` / `gcx use <env>` | Switch active shell to `<env>` |
| `gcx unset` / `gcx clear` | Return to global default `~/.config/gcloud` |
| `gcx ls` / `gcx list` | List all environments, projects, accounts, and ADC status |
| `gcx who` / `gcx status` | Show detailed status of the active environment |
| `gcx new <env> [flags]` | Create and authenticate a new isolated environment |
| `gcx rm <env> [-f]` | Delete an isolated environment directory |
| `gcx direnv <env>` | Output exports formatted for direnv `.envrc` |
| `gcx doctor` | Sanity check environment health, permissions, and ADC |
| `gcx help [cmd]` / `gcx <cmd> help` | Display built-in command help |

---

## Google Cloud Prerequisites

`gcx new` does more than write local config: it authenticates you, grants your own user account impersonation roles on the target project, and optionally creates a service account and binds roles to it.

### Required APIs

These must already be enabled on the target project. `gcx` does not enable them for you.

```
cloudresourcemanager.googleapis.com
iam.googleapis.com
iamcredentials.googleapis.com
```

### Required permissions

Your user account needs the following on the target project:

| Permission | Needed for |
| :--- | :--- |
| `resourcemanager.projects.get` | Resolving the target project |
| `resourcemanager.projects.getIamPolicy` | Checking existing role bindings |
| `resourcemanager.projects.setIamPolicy` | Granting impersonation roles |
| `iam.serviceAccounts.get` | Checking whether the service account exists |
| `iam.serviceAccounts.create` | Creating the service account |

The simplest predefined-role combination covering this set:

```
roles/resourcemanager.projectIamAdmin
roles/iam.serviceAccountCreator
```

The two `iam.serviceAccounts.*` permissions are only used when you pass a service account. The three `resourcemanager.*` permissions are required on **every** run, including `--no-impersonate`, because `gcx new` always grants your user account `roles/iam.serviceAccountTokenCreator` and `roles/iam.serviceAccountUser` on the project.

> [!WARNING]
> There is no read-only mode. A user with only `roles/viewer` cannot create an environment at all, because every `gcx new` run writes to the project IAM policy.

> [!WARNING]
> When you supply a service account, `gcx new` grants it `roles/editor` **and** `roles/resourcemanager.projectIamAdmin` on the project. That combination is close to owner: `projectIamAdmin` lets the service account grant itself almost any further role. These roles are currently hardcoded and not configurable by flag. Review this before pointing `gcx` at a production project.

---

## Quick Start & Examples

### Create a new environment
```bash
# Interactive wizard
gcx new prod-env

# Scripted with specific project and service account
gcx new prod-env --project my-gcp-project --sa deployer --impersonate
```

### Switch environments
```bash
gcx prod-env
# or
gcx use prod-env
```

### Check active identity
```bash
gcx who
```

### Return to default global gcloud
```bash
gcx unset
```

---

## Environment Variables

| Variable | Description |
| :--- | :--- |
| `GCX_HOME` / `GCLOUD_ENVS_DIR` | Root directory for isolated trees (default: `~/.gcloud-envs`) |
| `NO_COLOR` | Disable ANSI color output |

---

## License

[Apache 2.0](LICENSE)
