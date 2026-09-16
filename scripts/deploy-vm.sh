#!/bin/bash
set -e

VM_IP="34.75.25.242"
VM_USER="hongwu90"
SSH_KEY="$HOME/.ssh/gcp_key"

echo "======================================"
echo "🚀 Deploying Scout to VM: $VM_IP"
echo "======================================"

echo "[1/4] Building frontend..."
cd frontend
npm run build
cd ..

# Sync dist to backend
rm -rf backend/internal/web/dist/*
cp -r frontend/dist/* backend/internal/web/dist/
touch backend/internal/web/dist/.keep

echo "[2/4] Building Linux binary..."
cd backend
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ../scout-linux cmd/fusion/main.go
cd ..

echo "[3/4] Uploading new binary to VM (atomic replacement)..."
scp -i "$SSH_KEY" -o StrictHostKeyChecking=no scout-linux "$VM_USER@$VM_IP:/home/$VM_USER/scout-linux.new"

echo "[4/4] Activating new binary and restarting service..."
ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no "$VM_USER@$VM_IP" "chmod +x /home/$VM_USER/scout-linux.new && mv -f /home/$VM_USER/scout-linux.new /home/$VM_USER/scout-linux && sudo systemctl restart scout && sudo systemctl is-active scout"

echo "======================================"
echo "✅ Deployment complete! Scout is running active on http://$VM_IP"
echo "======================================"
