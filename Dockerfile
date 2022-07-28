FROM golang:alpine3.16 as build

RUN apk add --no-cache --update git
COPY go.sum go.mod /src/
WORKDIR /src
RUN go mod download
COPY cmd/mpa/*.go /src/cmd/mpa/
RUN go build -ldflags="-s -w" ./cmd/mpa

FROM alpine:3.16

COPY --from=build /src/mpa /usr/local/bin
ADD cmd/mpa/templates /mpa/templates
ADD cmd/mpa/static /mpa/static

WORKDIR /mpa
CMD mpa -f /data/mpa.db -http :4000
