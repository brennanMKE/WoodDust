#!/usr/bin/env bash
#
# provision.sh — one-time (and safely re-runnable) host setup for an
# environment on photon. Run it before the first scripts/deploy.sh <env>,
# and again whenever deploy/ (unit template, vhosts, envs) changes.
#
#   scripts/provision.sh preprod
#   scripts/provision.sh prod            # asks to confirm
#
# In order (each step is skipped when already done):
#   1. DNS (prod): Route 53 A records for the apex and www -> photon's EIP,
#      using the `admin` AWS profile from this Mac.
#   2. Host: system user, $ROOT/releases, Postgres role + $DB + ${DB}_identity,
#      $ETC/config.env with secrets generated ON the host (never printed,
#      never copied back; on a re-run, keys added since, e.g. SETTINGS_KEY,
#      are generated into the existing file), port in /etc/photon/ports.conf + `photon-ports
#      render`, the systemd unit from deploy/systemd/site.service.in, Apache
#      log directories.
#   3. Apache :80 vhosts, configtest, reload.
#   4. Certificate (prod): Let's Encrypt HTTP-01 via the shared webroot
#      /var/www/letsencrypt, with a deploy hook that reloads httpd.
#   5. Apache :443 vhosts, configtest, reload; `photon-ports check`.
#
# Follows photon's conventions (~/Developer/AWS: AddSite.md, AccessLogs.md,
# the port registry). Never edits another site's files; configtest always
# runs before a reload so a bad file cannot take the other sites down.

. "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

load_env "${1:-}"
[ "${2:-}" = --yes ] && ASSUME_YES=1
cd "$REPO"
confirm_prod

hdr "preflight ($ENV_NAME)"
ssh -o BatchMode=yes -o ConnectTimeout=10 "$SSH_HOST" true || die "cannot ssh to $SSH_HOST"
info "ssh        $SSH_HOST reachable"
for f in "$APACHE_DIR"/*.conf deploy/systemd/site.service.in deploy/config.env.example; do
  [ -f "$f" ] || die "missing $f"
done

# ── 1. DNS ──────────────────────────────────────────────────────────────────
if [ -n "${DNS_ZONE:-}" ]; then
  hdr "dns ($DNS_ZONE)"
  AWS=(aws --profile "$AWS_PROFILE_DNS" --region us-east-1)
  ZONE_ID="$("${AWS[@]}" route53 list-hosted-zones-by-name --dns-name "$DNS_ZONE." --max-items 1 \
      --query "HostedZones[?Name=='$DNS_ZONE.'].Id" --output text | sed 's#/hostedzone/##')"
  [ -n "$ZONE_ID" ] || die "no Route 53 hosted zone for $DNS_ZONE"
  IP="$("${AWS[@]}" ec2 describe-instances --instance-ids "$INSTANCE_ID" \
      --query 'Reservations[].Instances[].PublicIpAddress' --output text)"
  [ -n "$IP" ] && [ "$IP" != None ] || die "no public IP for $INSTANCE_ID"
  info "zone $ZONE_ID, photon $IP"
  BATCH="$(mktemp)"
  cat >"$BATCH" <<JSON
{ "Comment": "Dust Almanac ($ENV_NAME): apex and www to photon",
  "Changes": [
    { "Action": "UPSERT", "ResourceRecordSet": { "Name": "$DNS_ZONE.", "Type": "A", "TTL": 300,
        "ResourceRecords": [{ "Value": "$IP" }] } },
    { "Action": "UPSERT", "ResourceRecordSet": { "Name": "www.$DNS_ZONE.", "Type": "A", "TTL": 300,
        "ResourceRecords": [{ "Value": "$IP" }] } } ] }
JSON
  CHANGE="$("${AWS[@]}" route53 change-resource-record-sets --hosted-zone-id "$ZONE_ID" \
      --change-batch "file://$BATCH" --query ChangeInfo.Id --output text)"
  rm -f "$BATCH"
  "${AWS[@]}" route53 wait resource-record-sets-changed --id "$CHANGE"
  ok "$DNS_ZONE and www.$DNS_ZONE -> $IP (INSYNC)"
fi

# ── 2. Host ─────────────────────────────────────────────────────────────────
hdr "host"
STAGE="/tmp/$UNIT-provision"
ssh -o BatchMode=yes "$SSH_HOST" "rm -rf $STAGE && mkdir -p $STAGE/apache"
sed -e "s|@UNIT@|$UNIT|g" -e "s|@ENV_NAME@|$ENV_NAME|g" -e "s|@ROOT@|$ROOT|g" -e "s|@ETC@|$ETC|g" \
  deploy/systemd/site.service.in >"${TMPDIR:-/tmp}/$UNIT.service"
scp -q "${TMPDIR:-/tmp}/$UNIT.service" deploy/config.env.example "$SSH_HOST:$STAGE/"
scp -q "$APACHE_DIR"/*.conf "$SSH_HOST:$STAGE/apache/"
rm -f "${TMPDIR:-/tmp}/$UNIT.service"

remote "
STAGE=$STAGE
pg() { sudo -u postgres -H sh -c \"cd /tmp && \$*\"; }

id \$UNIT >/dev/null 2>&1 || { sudo useradd --system --home-dir \$ROOT --no-create-home --shell /sbin/nologin \$UNIT; echo '✓  user '\$UNIT; }
sudo install -d -m 0755 -o root -g root \$ROOT \$ROOT/releases
sudo install -d -m 0750 -o root -g \$UNIT \$ETC

rpm -q postgresql15-contrib >/dev/null 2>&1 || { sudo dnf install -y -q postgresql15-contrib; echo '✓  postgresql15-contrib'; }
pg \"psql -tAc \\\"select 1 from pg_roles where rolname='\$UNIT'\\\" | grep -q 1\" || { pg \"createuser \$UNIT\"; echo '✓  role '\$UNIT; }
for d in \$DB \${DB}_identity; do
  pg \"psql -tAc \\\"select 1 from pg_database where datname='\$d'\\\" | grep -q 1\" || { pg \"createdb -O \$UNIT \$d\"; echo '✓  database '\$d; }
done

if sudo test -f \$ETC/config.env; then
  echo '·· '\$ETC'/config.env exists; left as is apart from missing generated keys'
  # Keys added to config.env.example after a host was provisioned. Only keys
  # that are safe to create late belong here: never IDENTITY_BOOTSTRAP_CODE
  # (removing it is how enrollment is closed) or the identity secrets (a new
  # one would break sign-in). A value never appears in argv or the output:
  # it goes through a pipe from printf, a shell builtin.
  for k in SETTINGS_KEY; do
    sudo grep -Eq \"^\$k=.+\" \$ETC/config.env && continue
    sudo sed -i \"/^\$k=\$/d\" \$ETC/config.env
    printf '\n# Generated by provision.sh\n%s=%s\n' \$k \"\$(openssl rand -hex 32)\" | sudo tee -a \$ETC/config.env >/dev/null
    sudo chown root:\$UNIT \$ETC/config.env && sudo chmod 0640 \$ETC/config.env
    echo '✓  generated '\$k' in '\$ETC'/config.env (restart '\$UNIT' to use it)'
  done
else
  gen() { openssl rand -hex 32; }
  F=\$(mktemp)
  sed -e \"s|^BASE_URL=.*|BASE_URL=\$BASE_URL|\" \
      -e \"s|postgres://wooddust@/wooddust|postgres://\$UNIT@/\$DB|\" \
      -e \"s|^IDENTITY_CLIENT_ID=.*|IDENTITY_CLIENT_ID=\$UNIT|\" \
      -e \"s|^IDENTITY_CRYPTO_KEY=.*|IDENTITY_CRYPTO_KEY=\$(gen)|\" \
      -e \"s|^IDENTITY_CLIENT_SECRET=.*|IDENTITY_CLIENT_SECRET=\$(gen)|\" \
      -e \"s|^COOKIE_KEY=.*|COOKIE_KEY=\$(gen)|\" \
      -e \"s|^IDENTITY_BOOTSTRAP_CODE=.*|IDENTITY_BOOTSTRAP_CODE=\$(gen)|\" \
      -e \"s|^FORM_TOKEN_KEY=.*|FORM_TOKEN_KEY=\$(gen)|\" \
      -e \"s|^SETTINGS_KEY=.*|SETTINGS_KEY=\$(gen)|\" \
      -e \"s|^# /etc/wooddust/config.env|# \$ETC/config.env|\" \$STAGE/config.env.example >\$F
  sudo install -m 0640 -o root -g \$UNIT \$F \$ETC/config.env; rm -f \$F
  echo '✓  '\$ETC'/config.env (secrets generated on the host)'
fi

REG=/etc/photon/ports.conf
if grep -Eq \"^[0-9]+[[:space:]]+\$UNIT[[:space:]]\" \$REG; then
  have=\$(awk -v u=\$UNIT '\$2==u {print \$1}' \$REG)
  [ \"\$have\" = \"\$PORT\" ] || { echo \"✗  ports.conf has \$UNIT on \$have, env says \$PORT\" >&2; exit 1; }
  echo '·· port '\$PORT' already registered'
else
  if awk -v p=\$PORT '\$1==p' \$REG | grep -q .; then echo \"✗  port \$PORT is taken in ports.conf\" >&2; exit 1; fi
  sudo cp -p \$REG \$REG.bak-\$(date -u +%Y%m%dT%H%M%SZ)
  line=\$(printf '%-7s %-22s %-9s %s   # Dust Almanac (%s)' \$PORT \$UNIT loopback \"\$HOSTS\" \$ENV_NAME)
  last=\$(grep -nE '^811[0-9][[:space:]]' \$REG | tail -1 | cut -d: -f1)
  if [ -n \"\$last\" ]; then sudo sed -i \"\${last}a \$line\" \$REG; else echo \"\$line\" | sudo tee -a \$REG >/dev/null; fi
  echo '✓  registered port '\$PORT' for '\$UNIT
fi

sudo install -m 0644 \$STAGE/\$UNIT.service /etc/systemd/system/\$UNIT.service
sudo photon-ports render | tail -1
sudo systemctl enable \$UNIT >/dev/null 2>&1
sudo install -d -m 0700 /var/log/httpd/sites/\$LOG_HOST /var/log/httpd/redirects/http /var/log/httpd/redirects/https
sudo restorecon -R /var/log/httpd 2>/dev/null || true
sudo install -d -m 0755 /var/www/letsencrypt/.well-known/acme-challenge
echo '✓  unit, port drop-in, log directories'
"

# ── 3-5. Apache and certificate ─────────────────────────────────────────────
hdr "apache + certificate"
remote "
STAGE=$STAGE
CONF=/etc/httpd/conf.d
BK=\$STAGE/backup && mkdir -p \$BK
# install_confs <ssl|plain>: install that half of the vhosts, configtest, and
# only then reload. On a failed configtest, put back what was there before so
# the next reload by anyone still works.
install_confs() {
  local want=\$1 installed=''
  for f in \$STAGE/apache/*.conf; do
    case \$f in *-le-ssl.conf) [ \$want = ssl ] || continue ;; *) [ \$want = plain ] || continue ;; esac
    b=\$(basename \$f)
    if [ -f \$CONF/\$b ]; then
      if sudo cmp -s \$f \$CONF/\$b; then continue; fi
      sudo cp -p \$CONF/\$b \$BK/\$b
    fi
    sudo install -m 0644 -o root -g root \$f \$CONF/\$b
    installed=\"\$installed \$b\"
  done
  if [ -z \"\$installed\" ]; then echo \"·· \$want vhosts unchanged\"; return 0; fi
  if ! sudo apachectl configtest >/dev/null 2>&1; then
    sudo apachectl configtest 2>&1 | tail -5 >&2
    for b in \$installed; do
      if [ -f \$BK/\$b ]; then sudo cp -p \$BK/\$b \$CONF/\$b; else sudo rm -f \$CONF/\$b; fi
    done
    echo \"✗  configtest failed; reverted\$installed\" >&2; exit 1
  fi
  sudo systemctl reload httpd
  echo \"✓  \$want vhosts installed and httpd reloaded:\$installed\"
}

install_confs plain

if [ -n \"\$CERT_DOMAINS\" ] && ! sudo test -f /etc/letsencrypt/live/\$CERT_NAME/fullchain.pem; then
  args=''; for d in \$CERT_DOMAINS; do args=\"\$args -d \$d\"; done
  sudo certbot certonly -n --webroot -w /var/www/letsencrypt --cert-name \$CERT_NAME \$args \\
    --deploy-hook 'systemctl reload httpd'
  echo '✓  certificate '\$CERT_NAME
fi
sudo test -f /etc/letsencrypt/live/\$CERT_NAME/fullchain.pem || { echo \"✗  no certificate \$CERT_NAME\" >&2; exit 1; }

install_confs ssl
sudo photon-ports check | tail -1
rm -rf \$STAGE
"
ok "$ENV_NAME provisioned. Next: scripts/deploy.sh $ENV_NAME"
