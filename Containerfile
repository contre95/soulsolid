FROM golang:bookworm AS plugin-builder
ENV CGO_ENABLED=2
RUN apt-get update && apt-get install -y gcc libc-dev git
WORKDIR /plugins
RUN git clone https://github.com/contre95/soulsolid-dummy-plugin.git dummy
RUN cd dummy && go mod tidy && go build -buildmode=plugin -o /plugins/dummy.so main.go

FROM golang:bookworm AS app-builder
ENV CGO_ENABLED=2
RUN apt-get update && apt-get install -y gcc libc-dev libsqlite3-dev nodejs npm git
WORKDIR /app
# Copy your source code here - adjust the path as needed
ADD . .
RUN npm install && npm run build:assets
RUN go mod tidy
# Build without static linking for plugin compatibility
RUN go build -o /app/soulsolid src/main.go

FROM golang:bookworm
WORKDIR /app
ENV SS_VIEWS=/app/views
# Copy the dynamically built app and assets
COPY --from=app-builder /app/soulsolid /app/soulsolid
COPY --from=app-builder /app/views /app/views
COPY --from=app-builder /app/public /app/public
COPY --from=plugin-builder /plugins /app/plugins
ENTRYPOINT ["/app/soulsolid"]