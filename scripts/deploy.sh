#!/usr/bin/env bash
#
# deploy.sh — build a committed revision and make it live on photon.
#
#   scripts/deploy.sh preprod            # deploy HEAD to wooddust.sstools.co
#   scripts/deploy.sh prod               # deploy HEAD to www.dustalmanac.com (asks to confirm)
#   scripts/deploy.sh prod --ref v1.2    # any commit-ish
#   scripts/deploy.sh prod --yes         # no prompt (for automation)
#
# What it does:
#   1. Exports <ref> with `git archive` into a temp dir, so uncommitted work is
#      never shipped, and builds there: SPA (npm ci && build), seed -check,
#      then the linux/arm64 binary with the SPA embedded.
#   2. Uploads the archive + binary, unpacks to $ROOT/releases/<sha>/.
#   3. Points $ROOT/current at it and restarts the unit (ExecStartPre migrates
#      and reseeds from the release's data/).
#   4. Waits for /api/v1/health on loopback. On failure, switches back to the
#      previous release, restarts, and exits non-zero.
#   5. Registers the site's sign-in client on the first deploy only, prunes old
#      releases (keeps $KEEP_RELEASES and never the live/previous one), and
#      checks the public URL.
#
# Prod only deploys commits that are on origin/main.
# One-time host setup comes first: scripts/provision.sh <env>.

. "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

ENV_ARG="${1:-}"; shift || true
REF=HEAD
while [ $# -gt 0 ]; do
  case "$1" in
    --ref) REF="${2:?--ref needs a value}"; shift 2 ;;
    --yes) ASSUME_YES=1; shift ;;
    *) die "unknown argument: $1" ;;
  esac
done
load_env "$ENV_ARG"
cd "$REPO"

SHA="$(git rev-parse --short "$REF^{commit}")" || die "not a commit: $REF"
FULL="$(git rev-parse "$REF^{commit}")"

hdr "deploy $SHA to $ENV_NAME ($BASE_URL)"
git log -1 --format='·· %h %s (%an, %ar)' "$FULL"
if [ "$REF" = HEAD ] && [ -n "$(git status --porcelain --untracked-files=no)" ]; then
  warn "working tree has uncommitted changes; they are NOT part of this deploy"
fi
if [ "$ENV_NAME" = prod ]; then
  git fetch -q origin main
  git merge-base --is-ancestor "$FULL" origin/main || die "$SHA is not on origin/main; push it first"
fi
confirm_prod

hdr "build"
WORK="$(mktemp -d "${TMPDIR:-/tmp}/dustalmanac-build.XXXXXX")"
trap 'rm -rf "$WORK"' EXIT
git archive --format=tar "$FULL" | tar -x -C "$WORK"
git archive --format=tar.gz -o "$WORK/release.tgz" "$FULL"
( cd "$WORK/web" && npm ci --silent --no-audit --no-fund && npm run --silent build >/dev/null ) || die "SPA build failed"
mkdir -p "$WORK/web/dist" && touch "$WORK/web/dist/.gitkeep"
( cd "$WORK" && go run ./cmd/wooddust seed -check >/dev/null ) || die "data integrity check failed (go run ./cmd/wooddust seed -check)"
( cd "$WORK" && CGO_ENABLED=0 GOOS=linux GOARCH="$GOARCH_REMOTE" \
    go build -trimpath -ldflags "-s -w" -o "$WORK/wooddust" ./cmd/wooddust ) || die "go build failed"
ok "built $(du -h "$WORK/wooddust" | cut -f1) binary, $(du -h "$WORK/release.tgz" | cut -f1) release"

hdr "upload"
scp -q "$WORK/release.tgz" "$SSH_HOST:/tmp/$UNIT-$SHA.tgz"
scp -q "$WORK/wooddust" "$SSH_HOST:/tmp/$UNIT-$SHA.bin"
ok "uploaded"

hdr "activate"
remote "
SHA=$SHA
R=\$ROOT/releases/\$SHA
sudo test -f \$ETC/config.env || { echo 'not provisioned: run scripts/provision.sh '\$ENV_NAME >&2; exit 1; }
PREV=\$(readlink \$ROOT/current 2>/dev/null || true)
if [ \"\$PREV\" = \"\$R\" ]; then echo \"·· \$SHA is already live; restarting it\"; fi
sudo rm -rf \$R.new && sudo install -d -m 0755 \$R.new \$R.new/bin
sudo tar -xzf /tmp/\$UNIT-\$SHA.tgz -C \$R.new
sudo install -m 0755 /tmp/\$UNIT-\$SHA.bin \$R.new/bin/wooddust
sudo chown -R root:root \$R.new && sudo chmod -R go-w \$R.new
rm -f /tmp/\$UNIT-\$SHA.tgz /tmp/\$UNIT-\$SHA.bin
if [ \"\$PREV\" = \"\$R\" ]; then sudo rm -rf \$R.old && sudo mv \$R \$R.old; fi
sudo mv \$R.new \$R
sudo ln -sfn \$R \$ROOT/current
sudo systemctl restart \$UNIT || true
healthy=0
for i in \$(seq 1 30); do
  if curl -fsS -m 2 http://127.0.0.1:\$PORT/api/v1/health >/dev/null 2>&1; then healthy=1; break; fi
  sleep 1
done
if [ \$healthy != 1 ]; then
  echo '✗  health check failed; last log lines:' >&2
  sudo journalctl -u \$UNIT -n 25 --no-pager -o cat >&2 || true
  if [ -n \"\$PREV\" ] && [ \"\$PREV\" != \"\$R\" ] && [ -d \"\$PREV\" ]; then
    echo \"·· rolling back to \$PREV\" >&2
    sudo ln -sfn \$PREV \$ROOT/current && sudo systemctl restart \$UNIT || true
  elif [ -d \$R.old ]; then
    sudo rm -rf \$R && sudo mv \$R.old \$R && sudo systemctl restart \$UNIT || true
  fi
  exit 1
fi
sudo rm -rf \$R.old
echo \"✓  \$UNIT healthy: \$(curl -fsS http://127.0.0.1:\$PORT/api/v1/health)\"

# The site registers itself as its provider's client on every start
# (identity.New → RegisterClient), so there is no first-deploy step.

# Prune: keep the newest \$KEEP_RELEASES, never the live or previous one.
cd \$ROOT/releases
LIVE=\$(basename \"\$(readlink \$ROOT/current)\")
OLD=\$(basename \"\${PREV:-none}\")
ls -1t | grep -v -e '\\.new\$' -e '\\.old\$' | tail -n +\$((KEEP_RELEASES + 1)) | while read -r d; do
  [ \"\$d\" = \"\$LIVE\" ] || [ \"\$d\" = \"\$OLD\" ] || { sudo rm -rf -- \"\$d\"; echo \"·· pruned \$d\"; }
done
echo \"·· previous release: \${PREV:-none}\"
"

hdr "verify"
code="$(curl -s -o /dev/null -w '%{http_code}' "$BASE_URL/api/v1/health")"
[ "$code" = 200 ] || die "public health check returned $code for $BASE_URL"
ok "$BASE_URL is serving $SHA"
info "rollback: scripts/rollback.sh $ENV_NAME"
