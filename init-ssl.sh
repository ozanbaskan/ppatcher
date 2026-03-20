#!/bin/bash
# init-ssl.sh — Run once on the server to bootstrap Let's Encrypt SSL.
# Usage: bash init-ssl.sh your@email.com ppatcher.com
# Must be run from /opt/ppatcher

set -euo pipefail

EMAIL="${1:-}"
DOMAIN="${2:-ppatcher.com}"

if [ -z "$EMAIL" ]; then
  echo "Usage: bash init-ssl.sh your@email.com [domain]"
  exit 1
fi

echo "==> Starting db and webapp..."
docker compose -f docker-compose.prod.yml up -d db webapp

echo "==> Starting nginx in HTTP-only mode (for ACME challenge)..."
mkdir -p certbot/conf certbot/www

# Write a temporary HTTP-only nginx config
cat > nginx/nginx-bootstrap.conf << EOF
events { worker_connections 1024; }
http {
    server {
        listen 80;
        server_name $DOMAIN www.$DOMAIN;
        location /.well-known/acme-challenge/ {
            root /var/www/certbot;
        }
        location / {
            return 200 'bootstrapping ssl...';
            add_header Content-Type text/plain;
        }
    }
}
EOF

# Start nginx with bootstrap config
docker run -d --name nginx-bootstrap \
  -p 80:80 \
  -v "$(pwd)/nginx/nginx-bootstrap.conf:/etc/nginx/nginx.conf:ro" \
  -v "$(pwd)/certbot/www:/var/www/certbot:ro" \
  nginx:alpine

echo "==> Requesting Let's Encrypt certificate..."
docker compose -f docker-compose.prod.yml run --rm --entrypoint certbot certbot certonly \
  --webroot -w /var/www/certbot \
  -d "$DOMAIN" -d "www.$DOMAIN" \
  --email "$EMAIL" --agree-tos --no-eff-email

echo "==> Stopping bootstrap nginx..."
docker stop nginx-bootstrap && docker rm nginx-bootstrap

echo "==> Starting full stack with SSL..."
docker compose -f docker-compose.prod.yml up -d

echo ""
echo "✅ SSL setup complete! Visit https://$DOMAIN"
