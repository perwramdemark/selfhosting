# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repo is

A personal homelab configuration repository. It tracks Docker Compose stacks and supporting configuration for services running on a host named **houdini**. All container data lives under `/home/houdini/<service>/` on that host.

The external domain is `w8k.site`. Services are exposed via Cloudflare Tunnel → Nginx Proxy Manager, protected by Tinyauth (SSO via Pocket ID / OIDC).

## Repository layout

- `stacks/` — one `compose.yaml` per service or logical group (e.g. `servarr/` bundles Radarr, Sonarr, Lidarr, Bazarr, Prowlarr, Jellyseerr)
- `stacks/common-security.yml` / `stacks/common-security-lsio.yml` — shared hardening blueprints (see below)
- `glance/config/` — Glance dashboard YAML split into `glance.yml` (entrypoint), `home.yml`, `news.yml`, `bookmarks.yml` via `$include`
- `ev-charger/` — Go microservice that serves an HTML widget for the Glance dashboard, pulling EV charging start time and Nordpool electricity prices from Home Assistant
- `watchdog/` — Bash script + Dockerfile that polls a named container and restarts it via `docker start`, notifying via ntfy
- `scripts/` — maintenance scripts run as cron jobs on houdini; all send notifications via ntfy at `https://ntfy.w8k.site/scripts`
- Top-level service dirs (`adguard/`, `glance/`, `ntfy/`, etc.) — config files that are bind-mounted into containers; `data/` subdirs are gitignored

## Conventions

### Container hardening blueprints

Every stack `extends` one of two shared blueprints:

```yaml
extends:
  file: ../common-security.yml      # standard containers — sets user: 1002:1002
  service: hardened-blueprint

extends:
  file: ../common-security-lsio.yml # LinuxServer.io images — sets PUID/PGID=1002
  service: hardened-blueprint
```

Both set `TZ=Europe/Stockholm` and cap log files at 10 MB / 3 rotations. Apply the appropriate one when adding a new stack.

### Glance dashboard labels

Every container that should appear in the Glance dashboard needs these labels:

```yaml
labels:
  glance.name: Display Name
  glance.icon: sh:icon-slug        # selfh.st/icons slug
  glance.url: https://<service>.w8k.site
  glance.description: Short text
```

Child containers (e.g. Postgres, Redis) use `glance.parent: <parent-glance-id>` instead of a URL.

### Secrets and .env files

Sensitive values are never committed. Stacks reference them via `env_file: .env` or `${VAR}` substitution. The repo's `.gitignore` excludes `secrets.env`, `.env`, `*.key`, and all `data/` directories.

## ev-charger service

A minimal Go HTTP server (`ev-charger/main.go`) with two endpoints:
- `GET /ev-start-time-snippet` — returns an HTML fragment for a Glance custom widget, combining EV charging start time with a Nordpool price sparkline
- `GET /health`

Requires the `HA_TOKEN` env var (Home Assistant long-lived access token). Build and run:

```bash
cd ev-charger
go build -o ev-charger .
HA_TOKEN=<token> ./ev-charger
```

The stack in `stacks/ev-charger-start-time/compose.yml` builds from this directory.

## Scripts

All scripts assume they run on houdini and use ntfy for completion/failure notifications. Key scripts:
- `audit_containers.sh` — checks that no container runs as root
- `copy_backups_to_nas.sh` — copies app backup dirs to the NAS mount at `/srv/dev-disk-by-uuid-…`
- `ntfy-ssh-login.sh` — notifies on SSH login (drop in `/etc/profile.d/`)
- `paperless_document_exporter.sh` — triggers Paperless-ngx document export
- `syncdropbox.sh` / `syncgoogledrive.sh` — rclone sync wrappers
