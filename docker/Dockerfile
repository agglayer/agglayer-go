# CONTAINER FOR BUILDING BINARY
FROM golang:1.21 AS build

WORKDIR /go/src/github.com/0xPolygon/beethoven

# Install tools
RUN go install github.com/gobuffalo/packr/v2/packr2@v2.8.3

# Install modules
ADD go.mod go.sum ./
RUN go mod download

# Build binary
COPY . .

WORKDIR /go/src/github.com/0xPolygon/beethoven/db
RUN packr2

WORKDIR /go/src/github.com/0xPolygon/beethoven
RUN make build

# CONTAINER FOR RUNNING BINARY
FROM alpine:3.16.0

COPY --from=build /go/src/github.com/0xPolygon/beethoven/dist/beethoven /app/beethoven

EXPOSE 8444

ENTRYPOINT ["/app/beethoven"]
