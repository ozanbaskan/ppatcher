#!/bin/bash
# server-setup.sh — Run once on a fresh Oracle Linux / Ubuntu instance
# Requirements: Docker & Git must already be installed.
# Usage: sudo bash server-setup.sh

set -euo pipefail

DOMAIN="ppatcher.com"
APP_DIR="/opt/ppatcher"
REPO_URL="https://github.com/YOUR_ORG/YOUR_REPO.git"  # <-- update this

echo "==> Creating app directory..."
mkdir -p "$APP_DIR"
chown "$SUDO_USER":"$SUDO_USER" "$APP_DIR"

echo "==> Cloning repo..."
git clone "$REPO_URL" "$APP_DIR"
cd "$APP_DIR"

echo "==> Creating production .env file..."
cat > webapp/.env.prod << 'EOF'
DATABASE_URL=postgres://ppatcher:CHANGE_ME_DB_PASS@db:5432/ppatcher?sslmode=disable
GOOGLE_CLIENT_ID=CHANGE_ME
GOOGLE_CLIENT_SECRET=CHANGE_ME
APP_URL=https://ppatcher.com
SESSION_SECRET=CHANGE_ME_RANDOM_64_CHARS
PORT=8080
GA_MEASUREMENT_ID=G-C1Z99TE4B2
POSTGRES_PASSWORD=CHANGE_ME_DB_PASS
EOF
echo "  !! Edit webapp/.env.prod with real secrets before continuing !!"

echo "==> Creating certbot directories..."
mkdir -p certbot/conf certbot/www

echo "==> Opening firewall ports (ufw)..."
if command -v ufw &>/dev/null; then
    ufw allow 80/tcp
    ufw allow 443/tcp
    ufw allow 22/tcp
    ufw --force enable
fi

echo ""
echo "==> Next steps:"
echo ""
echo "  1. Edit /opt/ppatcher/webapp/.env.prod with your real secrets"
echo ""
echo "  2. Start nginx on port 80 only (for ACME challenge) and get the certificate:"
echo ""
echo "     cd /opt/ppatcher"
echo "     docker compose -f docker-compose.prod.yml up -d nginx certbot db webapp"
echo ""
echo "     # Then request the certificate:"
echo "     docker compose -f docker-compose.prod.yml run --rm certbot certonly \\"
echo "       --webroot -w /var/www/certbot \\"
echo "       -d $DOMAIN -d www.$DOMAIN \\"
echo "       --email your@email.com --agree-tos --no-eff-email"
echo ""
echo "  3. Reload nginx to pick up the certificate:"
echo "     docker compose -f docker-compose.prod.yml exec nginx nginx -s reload"
echo ""
echo "  4. Add GitHub Actions secrets (Settings → Secrets → Actions):"
echo "     DEPLOY_HOST      = <your oracle instance public IP>"
echo "     DEPLOY_USER      = ubuntu (or opc for Oracle Linux)"
echo "     DEPLOY_SSH_KEY   = <private SSH key — the public key must be in ~/.ssh/authorized_keys on the server>"
echo ""
echo "  5. Push to main branch — auto-deploy will trigger."
