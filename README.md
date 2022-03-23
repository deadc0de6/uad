[![Tests Status](https://github.com/deadc0de6/uad/workflows/tests/badge.svg)](https://github.com/deadc0de6/uad/actions)
[![License: GPL v3](https://img.shields.io/badge/License-GPL%20v3-blue.svg)](http://www.gnu.org/licenses/gpl-3.0)

[![Donate](https://img.shields.io/badge/donate-KoFi-blue.svg)](https://ko-fi.com/deadc0de6)

# uad

[uad](https://github.com/deadc0de6/uad) (**u**pload **a**nd **d**ownload) is a very tiny
web server allowing to upload and download files.

# Usage

```bash
Usage: ./uad [<options>] [<work-directory>]
  -debug
    	Debug mode
  -help
    	Show usage
  -host string
    	Host to listen to
  -max-upload string
    	Max upload size in bytes (default "1G")
  -no-downloads
    	Disable downloads
  -no-uploads
    	Disable uploads
  -port int
    	Port to listen to (default 6969)
  -show-hidden
    	Show hidden files
  -version
    	Show version
```

# Install

Pick a release from [the release page](https://github.com/deadc0de6/uad/releases) and
install it in your `$PATH` or [compile from source](#compile-from-source).

## Docker

A docker image is availabe on [dockerhub](https://hub.docker.com/r/deadc0de6/uad).
```bash
docker run -d --name uad -p 6969:6969 -v /tmp/uad-files:/files deadc0de6/uad
```

You can also build the image yourself:
```bash
docker build -t uad .
```

## Reverse proxy

nginx for a sub domain
```
location / {
  client_max_body_size 1G;
  proxy_pass       http://127.0.0.1:6969/;
}
```

nginx for a subpath at `/uad/` for example
```
location /uad/ {
  client_max_body_size 1G;
  proxy_pass       http://127.0.0.1:6969/;
}
```

If you specified a different `-max-upload` than the default (`1G`),
you need to adapt the `client_max_body_size`.

## Compile from source

```bash
$ go mod tidy
$ make
$ ./uad -help
```

# Screenshot

![](/screenshots/uad.png?raw=true "uad")

# Contribution

If you are having trouble using *uad*, open an issue.

If you want to contribute, feel free to do a PR.

# License

This project is licensed under the terms of the GPLv3 license.
