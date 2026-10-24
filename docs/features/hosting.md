---
weight: 120
title: "Hosting"
description: "The web server: port, health check, and using Soulsolid from scripts."
icon: "dns"
draft: false
toc: true
---

Soulsolid is a web app: a single server that serves the interface you use in your
browser and an [API](../api.md) you can call from scripts. This page covers the server
side of running it. For installing Soulsolid with Podman, Docker, Compose or Kubernetes,
see [Self hosted](../self-host.md).

## The web interface

By default Soulsolid listens on port **3535**. Open `http://<your-server>:3535` in a
browser to use it. To use a different port, set `server.port` in the config and
restart.

Soulsolid has **no built-in login**. Anyone who can reach the port can use the app,
including deleting music. If it's reachable from outside your home network, put it
behind a reverse proxy with authentication (or a VPN).

## Health check

`GET /health` answers `OK` while the server is running. Use it for container health
checks, Kubernetes liveness probes or uptime monitoring:

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 3535
```

## Using Soulsolid from scripts

Every action you can take in the interface can also be done over HTTP. When a request
doesn't come from the web interface, Soulsolid answers with **JSON** instead of a web
page, so you can use `curl` or your own scripts to search the library, start imports,
check jobs and more. See the [API reference](../api.md).

## Configuration

```yaml
server:
  port: 3535         # the port the web interface listens on
  show_routes: false # print every API route to the log on startup
```

Both settings only apply on startup, so restart Soulsolid after changing them.

## Related

- [Self hosted](../self-host.md): running Soulsolid in a container.
- [API reference](../api.md): every route and what it returns.
- [Logging](./logging.md): the server's own logs.
