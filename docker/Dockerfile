# CONTAINER FOR BUILDING BINARY
FROM golang:1.20 AS build

WORKDIR /go/src/github.com/0xPolygon/beethoven

# Install tools
RUN go install github.com/gobuffalo/packr/v2/packr2@v2.8.3

# Install modules
ADD go.mod go.sum ./
RUN go mod download

# Build binary
COPY . .
RUN cd ./db && packr2 && cd ../
RUN make build

# CONTAINER FOR RUNNING BINARY
FROM alpine:3.16.0

COPY --from=build /go/src/github.com/0xPolygon/beethoven/dist/beethoven /app/beethoven

EXPOSE 8444

CMD ["/bin/sh", "-c", "/app/beethoven run"]
