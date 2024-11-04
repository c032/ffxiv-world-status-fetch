FROM golang:1.23.2-alpine3.20 AS build

RUN addgroup -g '100000' 'golang' && adduser -D -G 'golang' -u '100000' 'golang'

USER golang
WORKDIR /home/golang
RUN mkdir -p app .cache/go

COPY . ./app
WORKDIR ./app

RUN \
	--mount=type=cache,target=/home/golang/.cache/go,uid=100000,mode=755\
	GOMODCACHE='/home/golang/.cache/go/gomodcache' \
	GOCACHE='/home/golang/.cache/go/gocache' \
	go build -v -x -mod=readonly -o ./ffxiv-world-status-fetch

FROM ghcr.io/c032/docker-alpine:3.20

USER root

COPY --from=build /home/golang/app/ffxiv-world-status-fetch /usr/local/bin/ffxiv-world-status-fetch

USER alpine

CMD ["/usr/local/bin/ffxiv-world-status-fetch"]
