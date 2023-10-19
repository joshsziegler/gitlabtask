#!/bin/bash
set -e # Stop execution if a command errors

# Install Nginx
sudo apt install nginx-core
sudo ufw allow 'Nginx HTTP' # Allow Nginx through the firewall
sudo rm sudo cp /etc/nginx/sites-enabled/default # Remove default site config
sudo cp zglr.org /etc/nginx/sites-available/     # Install my site config
sudo ln -s /etc/nginx/sites-available/zgl.org /etc/nginx/sites-enabled/ # Enable my site config
sudo nginx -t # Test nginx config
sudo systemctl reload nginx # Reload the server to take effect

# Install Let's Encrypt Certbot to aquire TLS certs
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx # Setup TLS certs for available domains (modifies Nginx config)

# Setup HTTP Basic Auth (if desired)
#sudo apt install apache2-utils
#sudo htpasswd -c /etc/apache2/.htpasswd josh
