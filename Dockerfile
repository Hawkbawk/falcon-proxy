FROM golang:1.16 as builder
WORKDIR /app

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY . .

RUN GOOS=linux GOARCH=amd64 go build -v -o auto-join-networks

FROM traefik:2.4

LABEL AUTHOR="Ryan Hawkins (ryanlarryhawkins@gmail.com)"

EXPOSE 80 443

# We avoid changing our directory so that we know which directory traefik
COPY --from=builder /app/auto-join-networks auto-join-networks
COPY traefik.yml .
COPY dynamic.yml .
COPY entrypoint.sh entrypoint.sh

ENTRYPOINT "/entrypoint.sh"

