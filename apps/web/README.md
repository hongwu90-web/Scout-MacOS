# 🌐 Scout Web App (Standalone Cloud Edition)

Welcome to the **Scout Web App** deployment package.

This edition is designed to run 24/7 on an always-on cloud server (such as Google Cloud Platform Always Free or AWS Free Tier) or your own home server.

---

## 🔑 Your Credentials

Your personal credentials are saved locally on your machine in:
`Project Scout/.env.webapp`

- **Username**: `alberto`
- **Password**: *(saved in `.env.webapp`)*

---

## 🚀 Option 1: Deploy on Google Cloud Platform (Always Free - Recommended)

Google Cloud offers an **e2-micro** virtual machine that is **100% free forever** (not just 12 months) in US regions (`us-central1`, `us-west1`, `us-east1`).

### Step 1: Create Free VM on Google Cloud
1. Go to [Google Cloud Console](https://console.cloud.google.com/).
2. Open **Compute Engine** > **VM Instances** > Click **Create Instance**.
3. Set the following options:
   - **Name**: `scout-web`
   - **Region**: `us-central1` (Iowa), `us-east1` (South Carolina), or `us-west1` (Oregon).
   - **Machine Configuration**: Series **E2**, Machine type **e2-micro** (2 vCPU, 1 GB memory — labelled *Free tier eligible*).
   - **Boot Disk**: Standard Persistent Disk, **30 GB** (Ubuntu 24.04 LTS or Debian 12).
   - **Firewall**: Check both **Allow HTTP traffic** and **Allow HTTPS traffic**.
4. Click **Create**.

### Step 2: Install Docker on the VM
Click **SSH** next to your VM instance in Google Cloud to open the terminal, then run:
```bash
# Update and install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER
newgrp docker
```

### Step 3: Run Scout on the Server
On your server terminal, create a directory and run Scout using your credentials:

```bash
mkdir -p scout && cd scout

# 1. Create your environment file with your credentials
cat << 'EOF' > .env
FUSION_USERNAME=alberto
FUSION_PASSWORD=YOUR_PASSWORD_HERE
FUSION_PORT=8080
FUSION_HOST=0.0.0.0
FUSION_DB_PATH=/data/fusion.db
CLOUD_MONITOR_ENABLED=false
EOF
```
*(Replace `YOUR_PASSWORD_HERE` with the password from your local `.env.webapp`)*

```bash
# 2. Launch Scout with Docker
docker run -d \
  --name scout-web \
  --restart unless-stopped \
  -p 80:8080 \
  -v scout_data:/data \
  --env-file .env \
  ghcr.io/0x2e/fusion:latest
```

### Step 4: Open in Your Browser
Visit your VM's **External IP address** in your web browser:
`http://<YOUR-VM-EXTERNAL-IP>`

Enter your username (`alberto`) and password, and Scout is online 24/7!

---

## ☁️ Option 2: Deploy on AWS (EC2 Free Tier)

AWS offers 750 hours/month of `t2.micro` or `t3.micro` free for the first 12 months.

1. Go to [AWS Console](https://aws.amazon.com/console/) > **EC2** > **Launch Instance**.
2. Select **Ubuntu 24.04 LTS** (Free tier eligible).
3. Instance type: **t2.micro** or **t3.micro**.
4. Security Group: Allow inbound **HTTP (port 80)** and **Custom TCP (port 8080)** from Anywhere (`0.0.0.0/0`).
5. SSH into the instance, install Docker, and run the identical `docker run` command shown above.

---

## 💻 Local Testing (Run with Docker Compose)

To test the web container locally before cloud deployment:

```bash
cd "Project Scout"
# Copy your local web credentials
cp .env.webapp apps/web/.env

# Start with Docker Compose
docker compose -f apps/web/docker-compose.yml up --build
```
Then visit `http://localhost:8080`.
