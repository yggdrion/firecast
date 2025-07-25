# firecast

https://marketplace.visualstudio.com/items?itemName=humao.rest-client

docker run -t --rm -v ${PWD}:/app -w /app golangci/golangci-lint:v2.2.2 golangci-lint run

## Setup

1. Copy `.env.example` to `.env` and configure your values:

   - `DOMAIN_NAME`: Your domain name (e.g., `firecast.example.com`)
   - `ACME_EMAIL`: Your email for Let's Encrypt certificates
   - Other configuration values as needed

2. Ensure your domain points to your server's IP address

## Deployment Options

# Run full stack (development)

docker-compose up -d

# Run server-side only with HTTPS proxy

docker-compose -f docker-compose.server.yml up -d

This will start:

- Traefik reverse proxy with automatic HTTPS (Let's Encrypt)
- Redis database
- Firecast server

The server will be available at:

- **HTTPS**: `https://your-domain.com:8443` (with automatic SSL certificates)
- **HTTP**: `http://your-domain.com:8000` (redirects to HTTPS)
- **Traefik dashboard**: `http://your-server-ip:8080`

**Note**: Uses custom ports (8000/8443) to avoid conflicts with other services on standard ports (80/443).

# Run client-side only (after creating network)

docker network create firecast-network
docker-compose -f docker-compose.client.yml up -d
