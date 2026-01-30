### Docker Deployment

Here's an example `docker-compose.yaml` file:

```yaml
services:
  soulsolid:
    container_name: soulsolid
    image: contre95/soulsolid:nightly
    restart: unless-stopped

    ports:
      - 3535:3535

    environment:
      TELEGRAM_TOKEN: your_telegram_bot_token_here
      CONFIG_PATH: /app/config/config.yaml

    volumes:
      - ./config:/app/config
      - ./logs:/app/logs
````

Alternatively, you can use Podman commands or Podman-kube pod YAMLs for deployment.

## Notifications

Soulsolid allows you to configure notifications for various events. These notifications are set up in the `config.yaml` file. Here are some examples:

