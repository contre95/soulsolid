---
weight: 110
title: "Configuration"
description: "The Settings page: changing your configuration from the app and backing up your library."
icon: "settings"
draft: false
toc: true
---

All of Soulsolid's settings live in a single `config.yaml` file. You can edit that file
directly, or change most settings from the **Settings** page in the app. This page
explains how the Settings page works; for every option in the file, see the
[Configuration reference](../config.md).

## What you can do

- **Change settings** from the app, without touching the config file.
- **View** your current `config.yaml`.
- **Download a backup** of your library database.

## First start

If there's no config file when Soulsolid starts, it creates one with sensible defaults,
so you can run it without any setup. It also creates your library and download folders
if they don't exist yet.

If the config file has a mistake (for example a required setting is missing, or it
refers to an environment variable that isn't set), Soulsolid **won't start** and tells
you what's wrong.

## The Settings page

The Settings page lets you change:

- your **download** folder,
- **import** options: move or copy, duplicate handling, *always queue*, *allow missing
  metadata*, FAT32-safe filenames,
- your **path templates** (see [Paths](../paths.md)),
- **metadata providers**: turn them on or off and set their API keys,
- **logging** and **job logging**.

Changes take effect right away and are saved to `config.yaml`.

Some settings are **read-only** in the app and can only be changed in the config file
(they're marked as such on the page). These are the ones that only apply when Soulsolid
starts, or that are risky to change while it's running:

- the library folder,
- the database location,
- downloader plugins and artwork settings,
- lyrics providers,
- the server port,
- whether the download folder watcher starts automatically,
- webhooks.

Edit `config.yaml` and restart Soulsolid to change them.

> Running Soulsolid with a read-only config file (common with containers and
> Kubernetes ConfigMaps)? Changes made on the Settings page still apply until the next
> restart, but can't be saved to the file. Soulsolid logs a warning when this happens.

## Keeping secrets out of the config file

API keys and tokens don't have to be written in `config.yaml`. You can have Soulsolid
read them from an environment variable (`!env_var`) or from a file (`!env_file`, handy
for Docker and Kubernetes secrets):

```yaml
metadata:
  providers:
    discogs:
      secret: !env_var DISCOGS_API_KEY
    acoustid:
      secret: !env_file ACOUSTID_CLIENT_KEY_FILE   # ACOUSTID_CLIENT_KEY_FILE=/run/secrets/acoustid
```

See [Configuration → Environment Variables](../config.md#environment-variables).

## Backing up the database

The Settings page has a button to **download your library database**. It contains
everything Soulsolid knows about your library: tracks, artists, albums, playlists and
tags.

> The backup **only includes the database**, not your music files. Back up your library
> folder separately.

## Related

- [Configuration reference](../config.md): every option in `config.yaml`.
- [Importing](../importing.md), [Paths](../paths.md), [Lyrics](../lyrics.md), [Jobs](../jobs.md): settings for each feature.
- [Self hosted](../self-host.md): where the config file lives in a container.
