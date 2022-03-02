FROM golang:1-alpine as builder

WORKDIR /go/src/github.com/Hawkbawk/falcon-proxy

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o auto-join-networks

FROM traefik:2.4

ENV WORKDIR=/usr/src/app

WORKDIR ${WORKDIR}

LABEL AUTHOR="Ryan Hawkins (ryanlarryhawkins@gmail.com)"

COPY --from=builder /go/src/github.com/Hawkbawk/falcon-proxy/auto-join-networks /usr/local/bin/auto-join-networks
COPY traefik.yml entrypoint.sh ./
COPY config/dynamic.yml ./config/dynamic.yml

EXPOSE 80 443 8080

CMD ["${WORKDIR}/entrypoint.sh"]
