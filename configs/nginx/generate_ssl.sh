#!/bin/bash

# Script to generate self-signed SSL certificates for development

SSL_DIR="/etc/nginx/ssl"
mkdir -p $SSL_DIR

# Generate private key
openssl genrsa -out $SSL_DIR/nginx.key 2048

# Generate certificate signing request
openssl req -new -key $SSL_DIR/nginx.key -out $SSL_DIR/nginx.csr -subj "/C=US/ST=CA/L=San Francisco/O=ActivityLog/CN=activity-log.local"

# Generate self-signed certificate
openssl x509 -req -days 365 -in $SSL_DIR/nginx.csr -signkey $SSL_DIR/nginx.key -out $SSL_DIR/nginx.crt

# Set proper permissions
chmod 600 $SSL_DIR/nginx.key
chmod 644 $SSL_DIR/nginx.crt

echo "SSL certificates generated successfully!"
echo "Certificate: $SSL_DIR/nginx.crt"
echo "Private Key: $SSL_DIR/nginx.key"