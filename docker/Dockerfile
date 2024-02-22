# CONTAINER FOR BUILDING BINARY
FROM golang:1.21 AS build

WORKDIR /go/src/github.com/0xPolygon/agglayer

# Install modules
ADD go.mod go.sum ./
RUN go mod download

# Build binary
COPY . .

WORKDIR /go/src/github.com/0xPolygon/agglayer
RUN make build

# CONTAINER FOR RUNNING BINARY
FROM alpine:3.16.0

COPY --from=build /go/src/github.com/0xPolygon/agglayer/dist/agglayer /app/agglayer

EXPOSE 8444

ENTRYPOINT ["/app/agglayer"]
