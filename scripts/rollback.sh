#!/usr/bin/env bash
#
# rollback.sh — point an environment back at an earlier release on photon.
#
#   scripts/rollback.sh preprod          # list releases, switch to the one before current
#   scripts/rollback.sh prod <sha>       # switch to a specific release (asks to confirm)
#   scripts/rollback.sh prod --list      # just list releases
#
# Restarting reseeds the published tables from that release's data/. Schema
# migrations are append-only and are not undone; every release runs against
# the newest schema it has seen.

. "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

ENV_ARG="${1:-}"; TARGET="${2:-}"
[ "${3:-}" = --yes ] && ASSUME_YES=1
load_env "$ENV_ARG"

hdr "releases on $ENV_NAME"
remote '
LIVE=$(readlink $ROOT/current || true)
for d in $(ls -1t $ROOT/releases); do
  mark="  "; [ "$ROOT/releases/$d" = "$LIVE" ] && mark="* "
  echo "$mark$d  $(date -r $ROOT/releases/$d "+%Y-%m-%d %H:%M")"
done'
[ "$TARGET" = --list ] && exit 0

confirm_prod
hdr "rollback"
remote "
TARGET='$TARGET'
LIVE=\$(basename \"\$(readlink \$ROOT/current)\")
if [ -z \"\$TARGET\" ]; then
  TARGET=\$(ls -1t \$ROOT/releases | grep -vx \"\$LIVE\" | head -1)
fi
[ -n \"\$TARGET\" ] && [ -d \$ROOT/releases/\$TARGET ] || { echo \"no such release: \$TARGET\" >&2; exit 1; }
sudo ln -sfn \$ROOT/releases/\$TARGET \$ROOT/current
sudo systemctl restart \$UNIT
for i in \$(seq 1 30); do curl -fsS -m 2 http://127.0.0.1:\$PORT/api/v1/health >/dev/null 2>&1 && break; sleep 1; done
curl -fsS http://127.0.0.1:\$PORT/api/v1/health >/dev/null || { echo '✗  unhealthy after rollback' >&2; sudo journalctl -u \$UNIT -n 20 --no-pager -o cat >&2; exit 1; }
echo \"✓  \$ENV_NAME now on \$TARGET (was \$LIVE)\"
"
