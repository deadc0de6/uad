[![License: GPL v3](https://img.shields.io/badge/License-GPL%20v3-blue.svg)](http://www.gnu.org/licenses/gpl-3.0)

# uad

[uad](https://github.com/deadc0de6/uad) is a very tiny web server allowing to upload and download files.
It supports drag-and-drop.

![Alt text](/screenshots/uad.png?raw=true)

# Usage

# Install

Pick a release from [the release page](https://github.com/deadc0de6/uad/releases) and
install it in your `$PATH`.

## docker

```bash
$ docker build -t uad .
$ docker run -it --name uad -v /tmp/test:/uploads -p 6969:6969 uad
```

## Compile from source

compile
```bash
$ go build
$ ./uad -help
```

# Contribution

If you are having trouble using *uad*, open an issue.

If you want to contribute, feel free to do a PR.

# License

This project is licensed under the terms of the GPLv3 license.
