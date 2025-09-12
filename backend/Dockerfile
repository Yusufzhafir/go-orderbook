# Build  
FROM golang:1.25-alpine AS build  
WORKDIR /app  

RUN apk add --no-cache build-base  

COPY backend ./backend

# Copy all
COPY . .  
RUN cd backend/ && go mod vendor && go mod tidy


# Build
WORKDIR /app/backend/cmd  
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux go build \
    -trimpath -ldflags='-s -w' \
    -o /app/orderbook main.go  

# Runtime  
FROM alpine:3.18  
WORKDIR /usr/local/bin

COPY /backend/.env .
COPY --from=build /app/orderbook .


HEALTHCHECK --interval=10s --timeout=3s --retries=10 \
  CMD curl -fsS http://127.0.0.1:8080/healthz || exit 1

EXPOSE 8080  
ENTRYPOINT ["/usr/local/bin/orderbook"]  
