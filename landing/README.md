# NoJS Landing Page

This is the landing page and demo for the NoJS framework, ready to deploy to Fly.io.

## Local Development

```bash
# From the landing directory
go run main.go
```

Visit http://localhost:8080 to see the landing page and http://localhost:8080/demo/chat for the chat demo.

## Deployment to Fly.io

1. Install the Fly CLI:
```bash
curl -L https://fly.io/install.sh | sh
```

2. Login to Fly:
```bash
fly auth login
```

3. Deploy the app:
```bash
# From the nojs root directory
cd /path/to/nojs
fly launch

# For subsequent deployments
fly deploy
```

The Dockerfile is in the root directory and builds the landing page with the integrated chat demo.

## Structure

- `main.go` - Landing page server and routes
- `demo/chat/` - Chat demo package
- `static/` - CSS and static assets
- `fly.toml` - Fly.io configuration
- `Dockerfile` - Container configuration

## Features

- Landing page showcasing NoJS framework
- Live chat demo with HTML streaming
- Zero JavaScript required
- Ready for production deployment