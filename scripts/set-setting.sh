#!/usr/bin/env bash
#
# set-setting.sh — store an operator setting (internal/settings) on an
# environment, e.g. the ShortLinks API token for poster links.
#
#   scripts/set-setting.sh preprod shortlinks_api_key      # prompts, input hidden
#   pbpaste | scripts/set-setting.sh prod shortlinks_api_key   # or from a pipe
#   scripts/set-setting.sh prod shortlinks_campaign        # plain settings too
#
# The value is read here with `read -rs` (or from stdin when it is a pipe),
# never from arguments or the environment, and travels over ssh stdin into
#   wooddust settings set <key> --stdin
# run as the service user with the service's environment (systemd-run, the
# same way deploy.sh runs `identity register-site`). The binary validates
# it, encrypts secrets under SETTINGS_KEY and prints only the hint
# (slk_AbCdEfGh…). Nothing is echoed or logged on either side.
#
# Then open /admin/settings and press Test connection.

. "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

load_env "${1:-}"
KEY="${2:-}"
[ "${3:-}" = --yes ] && ASSUME_YES=1
case "$KEY" in
  shortlinks_api_key|shortlinks_api_url|shortlinks_public_base|shortlinks_campaign) ;;
  "") die "usage: scripts/set-setting.sh <preprod|prod> <key>" ;;
  *) die "unknown setting '$KEY' (shortlinks_api_key, shortlinks_api_url, shortlinks_public_base, shortlinks_campaign)" ;;
esac

if [ -t 0 ]; then
  confirm_prod
  read -rs -p "Value for $KEY on $ENV_NAME (input hidden): " VALUE
  echo
else
  [ "$ENV_NAME" != prod ] || [ "${ASSUME_YES:-0}" = 1 ] || die "piping a value to prod needs --yes"
  IFS= read -r VALUE || true
fi
[ -n "$VALUE" ] || die "empty value; nothing changed (to remove a setting, run: wooddust settings clear $KEY on the host)"

hdr "set $KEY on $ENV_NAME"
# printf is a shell builtin, so the value never appears in a process list.
# $KEY is validated above; the remote command line holds no secret.
printf '%s\n' "$VALUE" | ssh -o BatchMode=yes "$SSH_HOST" \
  "sudo systemd-run --pipe --wait --quiet -p User=$UNIT -p EnvironmentFile=$ETC/config.env \
   -p WorkingDirectory=$ROOT/current $ROOT/current/bin/wooddust settings set $KEY --stdin" \
  || { unset VALUE; die "setting $KEY failed (see the message above)"; }
unset VALUE
ok "$KEY stored on $ENV_NAME. Next: $BASE_URL/admin/settings → Test connection"
