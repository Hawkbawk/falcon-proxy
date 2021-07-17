FROM golang:1.16 as builder
WORKDIR /app
COPY . .

RUN GOOS=linux GOARCH=amd64 go mod download
RUN GOOS=linux GOARCH=amd64 go build -v -o auto-join-networks

FROM traefik:2.4

LABEL AUTHOR="Ryan Hawkins (ryanlarryhawkins@gmail.com)"

EXPOSE 80 443

# We avoid changing our directory so that we know which
COPY --from=builder /app/auto-join-networks auto-join-networks
COPY traefik.yml .
COPY dynamic.yml .

CMD [ "/bin/sh", "entrypoint.sh" ]

