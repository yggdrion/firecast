# firecast

https://marketplace.visualstudio.com/items?itemName=humao.rest-client

docker run -t --rm -v ${PWD}:/app -w /app golangci/golangci-lint:v2.2.2 golangci-lint run

# Run full stack (development)

docker-compose up -d

# Run server-side only

docker-compose -f docker-compose.server.yml up -d

# Run client-side only (after creating network)

docker network create firecast-network

docker-compose -f docker-compose.client.yml up -d
