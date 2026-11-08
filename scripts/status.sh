#!/usr/bin/env bash
#
# status.sh — what is running where. Read-only.
#
#   scripts/status.sh preprod
#   scripts/status.sh prod
#   scripts/status.sh prod --logs [N]    # also the last N (default 50) service log lines

. "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

load_env "${1:-}"
LOGS=0; N=50
[ "${2:-}" = --logs ] && { LOGS=1; N="${3:-50}"; }

hdr "$ENV_NAME on $SSH_HOST"
remote "
echo \"unit     \$UNIT: \$(systemctl is-active \$UNIT 2>/dev/null || true), port \$PORT\"
echo \"release  \$(basename \"\$(readlink \$ROOT/current 2>/dev/null)\" 2>/dev/null || echo none)\"
echo \"health   \$(curl -fsS -m 3 http://127.0.0.1:\$PORT/api/v1/health 2>/dev/null || echo unreachable)\"
echo \"releases \$(ls -1t \$ROOT/releases 2>/dev/null | tr '\n' ' ')\"
if [ $LOGS = 1 ]; then echo; sudo journalctl -u \$UNIT -n $N --no-pager -o cat; fi
"
hdr "public"
printf '%-40s %s\n' "$BASE_URL/api/v1/health" "$(curl -s -o /dev/null -w '%{http_code}' "$BASE_URL/api/v1/health")"
