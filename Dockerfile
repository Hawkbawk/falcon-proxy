FROM traefik:2.4

LABEL MAINTAINER="Ryan Hawkins (ryanlarryhawkins@gmail.com)"

WORKDIR /usr/src/app

EXPOSE 80 443

COPY traefik.yml .
COPY dynamic.yml .

