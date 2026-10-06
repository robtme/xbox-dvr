# Xbox DVR

Xbox DVR uses [OpenXBL](https://xbl.io/)'s API to find and download your latest DVR clips and screenshots.

## Download

You can download pre-compiled binaries for macOS, Linux and Windows from
the [releases](https://github.com/robtme/xbox-dvr/releases) page.

Alternatively, you can run the binary from within the pre-built Docker image:

```shell
docker run ghcr.io/robtme/xbox-dvr:latest
```

## Run with Docker

```shell
docker run --rm -e apiKey=your-openxbl-api-key -e autoDelete=false -e savePath=/out -v ./xbox-dvr:/out robtme/xbox-dvr:latest sync
```

## Run with Docker Compose

```yaml
services:
  xbox-dvr:
    container_name: 'xbox-dvr'
    hostname: 'xbox-dvr'
    image: 'robtme/xbox-dvr:latest'
    volumes:
      - '/your/output/directory:/out'
    environment:
      - apiKey=your-openxbl-api-key
      - autoDelete=false
      - savePath=/out
    restart: no
```

## Build

To build this application, you'll need [Go](https://golang.org/) installed.

```shell
git clone https://github.com/robtme/xbox-dvr.git
cd xbox-dvr
make
```

This will output a binary called `xbox-dvr`. You can then move it or use it by running `./bin/xbox-dvr` (on Unix devices).

## Set API Key

1. Create an account with [OpenXBL](https://xbl.io/)
2. Verify your email
3. Create an API key
4. Run `xdvr config set apiKey your-api-key`

## Sync Latest DVR Content

To sync your latest DVR content, run:

`xdvr sync`

## Delete Content From XBL

To automatically delete clips from XBL once they've been downloaded, run:

`xdvr config set autoDelete true`

### Global Flags

| Flag           | Short | Description                                  |
|----------------|-------|----------------------------------------------|
| `--debug`      | `-d`  | Print debug logs (default false)             |
| `--prettyLog`  |       | Pretty print logs to console (default true)  |
| `--timeout`    | `-t`  | Timeout duration for downloads (default 60s) |

## License

This repository is licensed under the Apache License version 2.0.

Some of the project's dependencies may be under different licenses.
