FROM golang:bookworm AS app-builder
ENV CGO_ENABLED=2
RUN apt-get update && apt-get install -y gcc libc-dev libsqlite3-dev nodejs npm git
WORKDIR /app
# Copy your source code here - adjust the path as needed
ADD . .
RUN npm install && npm run build
RUN go mod tidy
# Build without static linking for plugin compatibility
RUN go build -o /app/soulsolid src/main.go

FROM golang:bookworm
ENV CGO_ENABLED=2
RUN apt-get update && apt-get install -y git tree
RUN mkdir -p /app/plugins
WORKDIR /app
# Copy the dynamically built app and assets
COPY --from=app-builder /app/soulsolid /app/soulsolid
COPY --from=app-builder /app/public /app/public
COPY go.mod /app/go.mod
COPY go.sum /app/go.sum
COPY views /app/views
COPY src /app/src
ENTRYPOINT ["/app/soulsolid"]
