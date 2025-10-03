FROM golang:bookworm AS builder

# Install Go and C dependencies
ENV CGO_ENABLED=2
RUN apt-get update && apt-get install -y gcc libc-dev libsqlite3-dev tree

# Install Node.js and npm
RUN apt-get install -y nodejs npm

# Copy application files
WORKDIR /app
ADD . .

# Build CSS
RUN npm install
RUN npm run build:css

# Build Go application
RUN go mod tidy
RUN go build -ldflags='-s -w -extldflags "-static"' -o /app/soulsolid src/main.go

FROM scratch
ARG IMAGE_TAG
ENV IMAGE_TAG=$IMAGE_TAG

LABEL maintainer="contre95"

WORKDIR /app

# Copy application assets
COPY ./views /app/views
COPY --from=builder /app/public /app/public
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/soulsolid /app/soulsolid

# Copy tree binary
COPY --from=builder /usr/bin/tree /usr/bin/tree
COPY --from=builder /lib/x86_64-linux-gnu /lib/x86_64-linux-gnu
COPY --from=builder /lib64 /lib64

ENTRYPOINT ["/app/soulsolid"]

