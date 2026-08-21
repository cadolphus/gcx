package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init <bash|zsh>",
	Short: "Output shell integration script for zsh or bash",
	Long: `Generate shell wrapper functions and auto-completions for Zsh or Bash.

Add to your ~/.zshrc:
  eval "$(gcx init zsh)"

Add to your ~/.bashrc:
  eval "$(gcx init bash)"`,
	Args: cobra.ExactArgs(1),
	Example: `  # In ~/.zshrc:
  eval "$(gcx init zsh)"

  # In ~/.bashrc:
  eval "$(gcx init bash)"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		shell := strings.ToLower(args[0])
		switch shell {
		case "zsh":
			fmt.Print(zshInitScript)
		case "bash":
			fmt.Print(bashInitScript)
		default:
			return fmt.Errorf("unsupported shell: %s. Supported shells are 'zsh' and 'bash'", shell)
		}
		return nil
	},
}

const zshInitScript = `# gcx shell integration for Zsh
gcx() {
  local subcommands=("ls" "list" "who" "current" "status" "new" "rm" "delete" "doctor" "init" "env" "direnv" "completion" "help" "--help" "-h" "version" "--version")

  if [[ "$#" -eq 0 ]]; then
    command gcx help
    return $?
  fi

  local first="$1"

  # Handle unset / clear / reset
  if [[ "$first" == "unset" || "$first" == "clear" || "$first" == "reset" ]]; then
    local out
    out="$(command gcx unset --eval 2>&1)" || { echo "$out" >&2; return 1; }
    eval "$out"
    echo "Switched to global gcloud default (~/.config/gcloud)"
    return 0
  fi

  # Handle 'use' or 'switch'
  if [[ "$first" == "use" || "$first" == "switch" ]]; then
    local target="$2"
    if [[ -z "$target" ]]; then
      echo "usage: gcx use <env>" >&2
      return 1
    fi
    local out
    out="$(command gcx env "$target" 2>&1)" || { echo "$out" >&2; return 1; }
    eval "$out"
    command gcx who
    return 0
  fi

  # Handle direct environment name (e.g. 'gcx my-env')
  if [[ "$#" -eq 1 && ! " ${subcommands[*]} " =~ " ${first} " ]]; then
    local out
    out="$(command gcx env "$first" 2>&1)"
    if [[ $? -eq 0 ]]; then
      eval "$out"
      command gcx who
      return 0
    fi
  fi

  command gcx "$@"
}

# Zsh completion
_gcx() {
  local -a subcmds
  subcmds=(
    'ls:List all Google Cloud environments'
    'list:List all Google Cloud environments'
    'who:Show active environment details'
    'current:Show active environment details'
    'status:Show active environment details'
    'use:Switch active environment'
    'unset:Return to default global gcloud'
    'clear:Return to default global gcloud'
    'new:Create a new isolated environment'
    'rm:Delete an environment'
    'delete:Delete an environment'
    'direnv:Direnv integration helper'
    'doctor:Diagnose environments health'
    'init:Output shell integration'
    'help:Help about any command'
  )

  local envs_dir="${GCX_HOME:-${GCLOUD_ENVS_DIR:-$HOME/.gcloud-envs}}"
  local -a env_names
  if [[ -d "$envs_dir" ]]; then
    env_names=("$envs_dir"/*(N/:t))
  fi

  if (( CURRENT == 2 )); then
    _describe -t commands 'gcx command' subcmds
    _describe -t environments 'gcx environment' env_names
  elif (( CURRENT == 3 )); then
    case "$words[2]" in
      use|switch|rm|delete|env|direnv)
        _describe -t environments 'gcx environment' env_names
        ;;
      init)
        _values 'shell' bash zsh
        ;;
      *)
        ;;
    esac
  fi
}
compdef _gcx gcx
`

const bashInitScript = `# gcx shell integration for Bash
gcx() {
  local subcommands="ls list who current status new rm delete doctor init env direnv completion help --help -h version --version"

  if [ "$#" -eq 0 ]; then
    command gcx help
    return $?
  fi

  local first="$1"

  # Handle unset / clear / reset
  if [ "$first" = "unset" ] || [ "$first" = "clear" ] || [ "$first" = "reset" ]; then
    local out
    out="$(command gcx unset --eval 2>&1)" || { echo "$out" >&2; return 1; }
    eval "$out"
    echo "Switched to global gcloud default (~/.config/gcloud)"
    return 0
  fi

  # Handle 'use' or 'switch'
  if [ "$first" = "use" ] || [ "$first" = "switch" ]; then
    local target="$2"
    if [ -z "$target" ]; then
      echo "usage: gcx use <env>" >&2
      return 1
    fi
    local out
    out="$(command gcx env "$target" 2>&1)" || { echo "$out" >&2; return 1; }
    eval "$out"
    command gcx who
    return 0
  fi

  # Handle direct environment name (e.g. 'gcx my-env')
  if [ "$#" -eq 1 ]; then
    case " $subcommands " in
      *" $first "*)
        command gcx "$@"
        return $?
        ;;
      *)
        local out
        out="$(command gcx env "$first" 2>&1)"
        if [ $? -eq 0 ]; then
          eval "$out"
          command gcx who
          return 0
        fi
        ;;
    esac
  fi

  command gcx "$@"
}

# Bash completion
_gcx_bash_complete() {
  local cur prev words cword
  _init_completion || return

  local subcmds="ls list who current status use unset clear reset new rm delete direnv doctor init help"
  local envs_dir="${GCX_HOME:-${GCLOUD_ENVS_DIR:-$HOME/.gcloud-envs}}"
  local envs=""
  if [ -d "$envs_dir" ]; then
    envs="$(ls -1 "$envs_dir" 2>/dev/null)"
  fi

  if [ "$cword" -eq 1 ]; then
    COMPREPLY=( $(compgen -W "${subcmds} ${envs}" -- "$cur") )
  elif [ "$cword" -eq 2 ]; then
    case "$prev" in
      use|switch|rm|delete|env|direnv)
        COMPREPLY=( $(compgen -W "${envs}" -- "$cur") )
        ;;
      init)
        COMPREPLY=( $(compgen -W "bash zsh" -- "$cur") )
        ;;
      *)
        ;;
    esac
  fi
}
complete -F _gcx_bash_complete gcx
`
