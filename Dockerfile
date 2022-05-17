## build
FROM golang:1.17-buster AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

RUN go build -o /myserver

## deploy
FROM redhat/ubi8-minimal:8.5
WORKDIR /
COPY --from=build /myserver /myserver
USER 1001
ENTRYPOINT ["/myserver"]
