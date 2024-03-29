[![Tests Status](https://github.com/deadc0de6/uad/workflows/tests/badge.svg)](https://github.com/deadc0de6/uad/actions)
[![License: GPL v3](https://img.shields.io/badge/License-GPL%20v3-blue.svg)](http://www.gnu.org/licenses/gpl-3.0)

[![Donate](https://img.shields.io/badge/donate-KoFi-blue.svg)](https://ko-fi.com/deadc0de6)

# uad

[uad](https://github.com/deadc0de6/uad) (**u**pload **a**nd **d**ownload)
is a very tiny web server allowing to upload and download files.

![](/screenshots/uad.png?raw=true "uad")

# Usage

```bash
Very tiny web server allowing to upload and download files.

Usage:
  uad [flags] <path>...

Flags:
  -d, --debug               Debug mode
  -P, --from-parent         Paths get their names from parent dir
  -h, --help                help for uad
  -A, --host string         Host to listen to
  -m, --max-upload string   Max upload size (default "1G")
  -D, --no-downloads        Disable downloads
  -U, --no-uploads          Disable uploads
  -p, --port int            Port to listen to (default 6969)
  -s, --serve-subs          Serve all directories found in <path>
  -H, --show-hidden         Show hidden files
  -v, --version             version for uad
```

Every cli switch can be set using environment variable with `UAD_` prefix.
For example setting `--show-hidden` would need `export UAD_SHOW_HIDDEN=true`.

# Install

Quick start:
```bash
## You need at least golang 1.16
$ go install -v github.com/deadc0de6/uad@latest
$ uad
```

Or pick a release from
[the release page](https://github.com/deadc0de6/uad/releases) and
install it in your `$PATH`

Or use the docker image availabe on
[dockerhub](https://hub.docker.com/r/deadc0de6/uad)
```bash
$ docker run -d --name uad -p 6969:6969 -v /tmp/uad-files:/files deadc0de6/uad
```

Or [compile it from source](#compile-from-source)

## Compile from source

```bash
$ go mod tidy
$ make
$ ./uad --help
```

# Reverse proxy

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

If you specified a different `--max-upload` than the default (`1G`),
you need to adapt the `client_max_body_size`.

# Contribution

If you are having trouble using *uad*, open an issue.

If you want to contribute, feel free to do a PR.

# License

This project is licensed under the terms of the GPLv3 license.
