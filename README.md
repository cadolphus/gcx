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

### With Go
```bash
go install github.com/cadolphus/gcx@latest
```

### From Source
```bash
git clone https://github.com/cadolphus/gcx.git
cd gcx
make install
```

---

## Shell Setup

Add **one line** to your shell configuration file:

### Zsh (`~/.zshrc`)
```zsh
eval "$(gcx init zsh)"
```

### Bash (`~/.bashrc`)
```bash
eval "$(gcx init bash)"
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
