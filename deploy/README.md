# Deploying Syndi

Replaces the old `diygod/rsshub` docker-compose stack on a personal host. Runs as a
systemd **user** service under the SSH user (`root`), single binary + badger
cache on disk.

## Layout on the host

```
/root/syndi/
├── syndi        # binary (make build, CGO_ENABLED=0)
├── config.yaml  # copy of repo config.yaml (port 1200, badger cache)
├── syndi.env    # credentials, chmod 600 (ZHIHU_COOKIES migrated from old compose)
└── data/cache/  # badger cache (created on first run)
```

## Steps

```bash
# 1. Build locally (linux/amd64)
CGO_ENABLED=0 make build

# 2. Ship to host
ssh <host> 'mkdir -p /root/syndi'
scp build/syndi config.yaml <host>:/root/syndi/
# then on <host>: fill /root/syndi/syndi.env from the old <path-to-old-rsshub-stack>/docker-compose.yml env

# 3. Install user service
ssh <host> 'mkdir -p ~/.config/systemd/user'
scp deploy/syndi.service <host>:.config/systemd/user/
ssh <host> 'systemctl --user daemon-reload && systemctl --user enable --now syndi'
ssh <host> 'loginctl enable-linger $USER'   # survive reboots without a login session
```

Feeds are served at `http://<host>:1200/rss/<route>` (note the `/rss` prefix;
the old RSSHub served `/<route>`).

## Listening / exposure

`server.host` (default `127.0.0.1`) controls the bind addresses; there is no
built-in auth. On a typical private deployment it is set to `127.0.0.1,172.17.0.1` — loopback for local
checks plus the docker0 gateway so containers (FreshRSS via
`host.docker.internal`) can reach it. The instance is therefore not exposed on
the public interface regardless of cloud security-group rules.

## FreshRSS cutover

FreshRSS ran in docker and reached RSSHub via the compose network alias
`http://rsshub:1200/<route>`. A host systemd service is not on that network, so:

1. `<path-to-freshrss-compose>/docker-compose.yml`: drop the external `rsshub_net`, add
   `extra_hosts: ["host.docker.internal:host-gateway"]`.
2. Feed URLs in the FreshRSS sqlite DB:
   `http://rsshub:1200/<route>` → `http://host.docker.internal:1200/rss/<route>`
   (backup the db first; keep any `#force_feed` suffix).
3. `cd <path-to-old-rsshub-stack> && docker compose down` (files kept as reference).
