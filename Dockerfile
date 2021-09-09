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

# We avoid changing our directory so that we know which directory traefik
COPY --from=builder /go/src/github.com/Hawkbawk/falcon-proxy/auto-join-networks /usr/local/bin/auto-join-networks
COPY traefik.yml dynamic.yml entrypoint.sh ./

EXPOSE 80 8080

CMD "${WORKDIR}/entrypoint.sh"
