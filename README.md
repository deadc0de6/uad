[![License: GPL v3](https://img.shields.io/badge/License-GPL%20v3-blue.svg)](http://www.gnu.org/licenses/gpl-3.0)

# uad

[uad](https://github.com/deadc0de6/uad) (**u**pload **a**nd **d**ownload) is a very tiny
web server allowing to upload and download files.

# Usage

```bash
$ ./uad -help

Usage: ./uad [<options>]
  -help
    	Show usage
  -host string
    	Host to listen to
  -max-upload string
    	Max upload size in bytes (default "1.0G")
  -no-downloads
    	Disable downloads
  -no-uploads
    	Disable uploads
  -path string
    	Destination path for uploaded files (default "./uploads")
  -port int
    	Port to listen to (default 6969)
  -version
    	Show version
```

# Install

Pick a release from [the release page](https://github.com/deadc0de6/uad/releases) and
install it in your `$PATH`.

## docker

```bash
$ docker run --name uad -v /tmp/uploads:/uploads -p 6969:6969 deadc0de6/uad:v0.1
```

or built the image yourself

```bash
$ docker build -t uad .
$ docker run -it --name uad -v /tmp/uploads:/uploads -p 6969:6969 uad
```

## Compile from source

```bash
$ GO111MODULE=on go build
$ ./uad -help
```

# Screenshot

![](/screenshots/uad.png?raw=true "uad")

# Contribution

If you are having trouble using *uad*, open an issue.

If you want to contribute, feel free to do a PR.

# License

This project is licensed under the terms of the GPLv3 license.
