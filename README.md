# falcon-proxy

The falcon-proxy image is effectively a pre-configured Traefik container that uses the Docker
Traefik provider to automatically forward requests to the appropriate container. When combined
with some fun networking magic, this allows you to access any of your Docker web development
projects using a more typical TLD like "canvas.docker".

## Configuration

Currently, there is no configuration available for the container, as the falcon project as a
whole is still very much in early alpha. I do have plans to ensure that you can edit the
Traefik configuration files at your leisure in the future. This way, power-users can make any
tweaks they want to the config files, and the falcon CLI itself can support SSL.

## Contributing

Feel free to fork the project, make your changes, and open up a pull request! Just remember to be
kind to all involved, and remember this project is developed with my own free time.
