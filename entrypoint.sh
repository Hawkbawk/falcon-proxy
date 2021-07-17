#!/bin/sh

# Join networks in the background and keep Traefik logging in the foreground
/auto-join-networks &
/traefik