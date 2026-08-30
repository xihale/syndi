# HTTP Client Configuration

This document describes how `client` settings in `config.yaml` are applied at runtime.

## Fields

```yaml
client:
  user_agent: "Syndi/0.0.1 (+https://github.com/xihale/syndi)"
  timeout: 30s
  max_redirects: 10
  proxy: ""
  no_proxy: false
```

- `user_agent`: default `User-Agent` header for outgoing requests.
- `timeout`: total timeout per request.
- `max_redirects`: max number of redirects followed by the HTTP client.
- `proxy`: explicit proxy URL (for example `http://127.0.0.1:7890`).
- `no_proxy`: when `true`, disables all proxy usage.

## Effective Priority

Proxy settings are applied in this order:

1. `no_proxy: true` -> disable proxy entirely.
2. `proxy` is non-empty -> use configured proxy URL.
3. otherwise -> use environment proxy (`HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY`) via Go's default behavior.

## Retry Behavior

The client retries only when both conditions are met:

1. The HTTP method is idempotent (`GET`, `HEAD`, `OPTIONS`, `TRACE`, `PUT`, `DELETE`).
2. The response status is in retry whitelist: `408`, `425`, `429`, `500`, `502`, `503`, `504`.

Notes:

- Non-idempotent methods (for example `POST`) are not retried.
- When status is `429` or `503`, `Retry-After` header is respected if present.
