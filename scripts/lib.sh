# Shared helpers for scripts/{provision,deploy,rollback,status}.sh. Sourced, not run.
#
# Environments are defined in deploy/envs/<env>.env (preprod, prod). All of
# them live on photon (AWS docs: ~/Developer/AWS, AddSite.md, Migration.md).

set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SSH_HOST="${SSH_HOST:-photon}"
INSTANCE_ID="${INSTANCE_ID:-i-01c45429c78f3adf7}"   # photon (web-2026)
AWS_PROFILE_DNS="${AWS_PROFILE_DNS:-admin}"         # can write Route 53
GOARCH_REMOTE=arm64                                 # photon is aarch64

if [ -t 1 ]; then C_B=$'\e[34m' C_G=$'\e[32m' C_Y=$'\e[33m' C_R=$'\e[31m' C_0=$'\e[0m'; else C_B= C_G= C_Y= C_R= C_0=; fi
hdr()  { printf '\n%s━━ %s ━━%s\n' "$C_B" "$*" "$C_0"; }
info() { printf '·· %s\n' "$*"; }
ok()   { printf '%s✓%s  %s\n' "$C_G" "$C_0" "$*"; }
warn() { printf '%s!%s  %s\n' "$C_Y" "$C_0" "$*" >&2; }
die()  { printf '%s✗  %s%s\n' "$C_R" "$*" "$C_0" >&2; exit 1; }

# load_env <preprod|prod>: sources deploy/envs/<env>.env into the shell.
load_env() {
  local env="${1:-}"
  [ -n "$env" ] || die "environment required: preprod or prod"
  local f="$REPO/deploy/envs/$env.env"
  [ -f "$f" ] || die "unknown environment '$env' (no $f)"
  # shellcheck disable=SC1090
  . "$f"
}

# remote <script>: runs a bash script on the host as ec2-user (sudo inside).
# The env file's variables are passed in, so remote scripts can use them.
remote() {
  local vars
  vars="$(printf 'ENV_NAME=%q UNIT=%q DB=%q ROOT=%q ETC=%q PORT=%q BASE_URL=%q HOSTS=%q LOG_HOST=%q CERT_NAME=%q CERT_DOMAINS=%q KEEP_RELEASES=%q' \
    "$ENV_NAME" "$UNIT" "$DB" "$ROOT" "$ETC" "$PORT" "$BASE_URL" "$HOSTS" "$LOG_HOST" "$CERT_NAME" "$CERT_DOMAINS" "$KEEP_RELEASES")"
  ssh -o BatchMode=yes "$SSH_HOST" "cd /tmp && $vars bash -s" <<<"set -euo pipefail
$1"
}

# confirm_prod: production changes need a typed confirmation unless --yes.
confirm_prod() {
  [ "$ENV_NAME" = prod ] || return 0
  [ "${ASSUME_YES:-0}" = 1 ] && return 0
  [ -t 0 ] || die "refusing to change prod non-interactively without --yes"
  local reply
  read -r -p "This changes PRODUCTION ($BASE_URL). Type 'prod' to continue: " reply
  [ "$reply" = prod ] || die "aborted"
}
