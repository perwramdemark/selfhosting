# 🏡 My Homelab Setup

This repository documents the containers running in my personal homelab. It includes services for media management, networking, security, notifications, and user interface improvements. All services are containerized and orchestrated using Docker.

---

## 🔧 Infrastructure & Management Tools

### **[Dockge](https://github.com/louislam/dockge)**

* **Image**: `louislam/dockge`
* **Purpose**: A web-based Docker Compose stack manager. Helps visualize and control multiple Docker stacks easily.

### **[Diun](https://crazymax.dev/diun/)**

* **Image**: `crazymax/diun`
* **Purpose**: Docker image update notifier. Sends alerts when container images are updated.

### **[Cloudflared](https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/)**

* **Image**: `cloudflare/cloudflared`
* **Purpose**: Exposes internal services to the public securely using Cloudflare Tunnel, bypassing the need for port forwarding.

---

## 🔐 Network & Security

### **[AdGuard Home](https://github.com/AdguardTeam/AdGuardHome)**

* **Image**: `adguard/adguardhome`
* **Purpose**: DNS-level ad and tracker blocking. Acts as a local DNS server and provides privacy filtering.

---

## 📬 Notifications & Messaging

### **[ntfy](https://ntfy.sh/)**

* **Image**: `binwiederhier/ntfy`
* **Purpose**: Push notification service for sending messages to devices using simple HTTP.

---

## 🗣️ Communication

### **[Beszel](https://github.com/henrygd/beszel)**

* **Image**: `henrygd/beszel` & `henrygd/beszel-agent`
* **Purpose**: Secure self-hosted chat app and its accompanying agent service for encrypted messaging.

---

## 🎬 Media & Entertainment Stack

### **[Jellyfin](https://jellyfin.org/)**

* **Image**: `jellyfin/jellyfin`
* **Purpose**: Media server for streaming movies, TV shows, music, and more.

### **[Jellyseerr](https://github.com/Fallenbagel/jellyseerr)**

* **Image**: `fallenbagel/jellyseerr`
* **Purpose**: Request management for Jellyfin users. Allows family/friends to request new content.

### **[Radarr](https://radarr.video/)**

* **Image**: `lscr.io/linuxserver/radarr`
* **Purpose**: Automated movie downloading via BitTorrent or Usenet.

### **[Sonarr](https://sonarr.tv/)**

* **Image**: `lscr.io/linuxserver/sonarr`
* **Purpose**: TV series management and automated downloading.

### **[Bazarr](https://www.bazarr.media/)**

* **Image**: `lscr.io/linuxserver/bazarr`
* **Purpose**: Automatically downloads subtitles for use with Radarr and Sonarr.

### **[Prowlarr](https://github.com/Prowlarr/Prowlarr)**

* **Image**: `lscr.io/linuxserver/prowlarr`
* **Purpose**: Indexer manager that integrates with Radarr, Sonarr, and others.

### **[Readarr](https://readarr.com/)**

* **Image**: `lscr.io/linuxserver/readarr:develop`
* **Purpose**: Manages and automates downloading of ebooks and audiobooks.

### **[Transmission with OpenVPN](https://github.com/haugene/docker-transmission-openvpn)**

* **Image**: `haugene/transmission-openvpn`
* **Purpose**: Torrent client with built-in VPN support for secure downloads.

---

## 🌐 Web & Proxy

### **[Nginx Proxy Manager](https://nginxproxymanager.com/)**

* **Image**: `jc21/nginx-proxy-manager`
* **Purpose**: Reverse proxy with a simple web UI for managing proxy hosts and SSL certs.

---

## 📊 Dashboard

### **[Homarr](https://github.com/ajnart/homarr)**

* **Image**: `ghcr.io/ajnart/homarr`
* **Purpose**: A sleek and customizable homepage/dashboard to access and organize all homelab services.

---

## 📁 Notes

* All services are designed to be accessible securely and locally.
* Port mappings are configured to avoid conflicts and expose services as needed.
* Services are monitored and auto-updated where supported (e.g., via Diun or Watchtower if used).

---
