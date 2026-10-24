---
weight: 10
title: "Self hosted"
description: "Run Soulsolid with Podman, Docker, Compose or Kubernetes."
icon: "article"
date: "2025-09-09T21:10:44+02:00"
lastmod: "2025-09-13T12:00:00+02:00"
draft: false
toc: true
---

## Container Usage

The application can run without copying `config.yaml` into the container. If no config file exists, it will automatically create one with sensible defaults.

### Environment Variable Support

Soulsolid supports environment variables in configuration files using the `!env_var` tag:

```yaml
metadata:
  providers:
    discogs:
      secret: !env_var DISCOGS_API_KEY
```

Secrets can also be loaded from a file (handy for Docker/Kubernetes secrets) with the
`!env_file` tag: the named environment variable holds the *path* to the file, and the
file's contents become the value. See [Configuration](config.md) for details.

The application will fail to start if a referenced environment variable is not set.

### Podman / Docker

```bash
# Build the image (or use the published contre95/soulsolid image)
podman build -t soulsolid .
# Create folders
mkdir music downloads logs data config
# Run with docker/podman
podman run -d --name soulsolid -p 3535:3535 \
  -v ./music:/app/music \
  -v ./downloads:/app/downloads \
  -v ./logs:/app/logs \
  -v ./data:/data \
  -v ./config:/config contre95/soulsolid:latest
```

The web interface will be available at `http://localhost:3535`.

The container works from `/app`, so the generated default config stores the library in
`/app/music`, downloads in `/app/downloads` and job logs in `/app/logs/jobs`. The database
defaults to `/app/library.db`; to keep it on the `./data` volume, set
`database.path: /data/library.db` in `config/config.yaml`.

### Compose

Here's an example `compose.yaml` file:

```yaml
services:
  soulsolid:
    image: contre95/soulsolid:latest
    ports:
      - "3535:3535"
    volumes:
      - ./config:/config
      - ./data:/data
      - ./logs:/app/logs
      - ./music:/app/music
      - ./downloads:/app/downloads
    environment: # optional, only if your config.yaml references them
      - DISCOGS_API_KEY=your_discogs_key_here
    restart: unless-stopped
```

### Podman kube / Kubernetes

Alternatively, you can use Podman-kube pod YAMLs for deployment:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: soulsolid
  labels:
    app: soulsolid
spec:
  restartPolicy: OnFailure
  containers:
    - name: soulsolid
      image: contre95/soulsolid:latest
      ports:
        - containerPort: 3535
          hostPort: 3535
      volumeMounts:
        - mountPath: /config/config.yaml
          name: config
        - mountPath: /music
          name: music
        - mountPath: /downloads
          name: downloads
  volumes:
    - name: music
      hostPath:
        path: /path/to/soulsolid_music
        type: DirectoryOrCreate
    - name: downloads
      hostPath:
        path: /path/to/soulsolid_downloads
        type: DirectoryOrCreate
    - name: config
      hostPath:
        path: ./config/config.yaml
        type: File
```

When mounting your own `config.yaml`, point `libraryPath` and `downloadPath` at the
mounted directories (`/music` and `/downloads` above).

For a Kubernetes `Deployment` + `ConfigMap`, see the
[Kubernetes examples](https://github.com/contre95/soulsolid/tree/main/docs/examples/kubernetes).

## Logs

Soulsolid logs to the container's output, so `podman logs soulsolid` (or
`docker logs soulsolid`) shows what it's doing. Each background job also writes its own
log, by default under `/app/logs/jobs`, which you can read from the **Jobs** section of
the app. See [Logging](features/logging.md).

## Health check

`GET /health` returns `OK` while Soulsolid is running, for container health checks or
Kubernetes probes. See [Hosting](features/hosting.md).

## Development

Want to run Soulsolid from source or contribute? See the
[Development](https://github.com/contre95/soulsolid#development) section of the README and
the [contributing guide](https://github.com/contre95/soulsolid/blob/main/CONTRIBUTING.md).
