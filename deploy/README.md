# Deploying Syndi

A minimal deployment recipe: single binary + badger cache on disk, runs as a
systemd **user** service under any SSH user (`<host>` below is your server,
e.g. an entry in `~/.ssh/config`).

## Layout on `<host>`

```
/root/syndi/
├── syndi        # binary (make build, CGO_ENABLED=0)
├── config.yaml  # local copy of config.example.yaml (port 1200, badger cache)
├── syndi.env    # credentials, chmod 600 (namespace credential env vars)
└── data/cache/  # badger cache (created on first run)
```

## Steps

```bash
# 1. Build locally (linux/amd64)
CGO_ENABLED=0 make build

# 2. Ship to <host>
cp config.example.yaml config.yaml   # edit if needed
ssh <host> 'mkdir -p /root/syndi'
scp build/syndi config.yaml <host>:/root/syndi/
# then on <host>: create /root/syndi/syndi.env from your credentials

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
built-in auth. A typical private setup is `127.0.0.1,172.17.0.1` — loopback for
local checks plus the docker0 gateway so containers (FreshRSS via
`host.docker.internal`) can reach it. The instance is therefore not exposed on
the public interface regardless of cloud security-group rules.

## Public access

If the instance must be reachable from the public internet, do **not** bind
`0.0.0.0` — put a reverse proxy in front of the loopback listener and put auth
at the proxy. Syndi's built-in `middleware.access_key` is enforced when set:
every request must then present the key via the `key` query parameter or the
`X-Access-Key` header (constant-time compare), with only `/status` exempt —
note this also gates the docs UI and `/api/*`. It is a single shared key
(keys passed as a query parameter can leak into logs), so HTTP Basic Auth at
the proxy remains the recommended setup for public instances. Feed readers
support it natively (FreshRSS: feed URL
`https://user:pass@syndi.example.com/rss/<route>`).

Caddy example for a public host (`rss.example.com`):

```caddyfile
rss.example.com {
	basic_auth {          # Caddy < 2.7 spells this directive "basicauth"
		syndi <bcrypt-hash>   # caddy hash-password
	}
	reverse_proxy 127.0.0.1:1200
}
```

```bash
caddy hash-password --plaintext '...'          # bcrypt hash for the block above
caddy validate --config /etc/caddy/Caddyfile   # then
systemctl reload caddy                         # zero-downtime reload
curl -u syndi:... https://rss.example.com/rss/zhihu/hot
```

## Migrating a dockerized RSSHub (e.g. FreshRSS) to Syndi

If your feed reader runs in docker and previously reached RSSHub through a
compose network alias `http://rsshub:1200/<route>`, a host systemd service is
not on that network, so:

1. In the reader's `docker-compose.yml`: drop the external `rsshub_net`, add
   `extra_hosts: ["host.docker.internal:host-gateway"]`.
2. Rewrite feed URLs in the reader's database:
   `http://rsshub:1200/<route>` → `http://host.docker.internal:1200/rss/<route>`
   (backup the db first; keep any `#force_feed` suffix).
3. Stop the old stack (`docker compose down`) once feeds validate; keep files
   around as reference until the cutover is confirmed.
