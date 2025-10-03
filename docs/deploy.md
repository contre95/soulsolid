### Docker Deployment

While there isn't a dedicated Docker deployment guide, you can use the provided `Dockerfile` to build a Docker image for Soulsolid.

Here's an example `docker-compose.yaml` file:

```yaml
version: "3.8"
services:
  soulsolid:
    image: soulsolid:latest
    ports:
      - "3535:3535"
    volumes:
      - ./config.yaml:/app/config.yaml
      - ./library.db:/app/library.db
      - ./logs:/app/logs
    environment:
      - TELEGRAM_TOKEN=your_telegram_bot_token_here
    restart: unless-stopped
````

Alternatively, you can use Podman commands or Podman-kube pod YAMLs for deployment.

## Notifications

Soulsolid allows you to configure notifications for various events. These notifications are set up in the `config.yaml` file. Here are some examples:

