# GitHub Enterprise Repository Synchroniser in Go

This application synchronises repositories between two GitHub Enterprise Server instances.
It uses GitHub Apps for authentication on both the source instance and the target instance.
For each configured repository it performs a bare clone from the source instance, then a mirror push to the target instance.

The code is structured as a small Go module with an internal package layout.
A Makefile is provided for common tasks, and a Docker image and compose file are supplied to run sync jobs in a container environment.

## Configuration

The application expects a `config.yaml` file with this structure.

```yaml
github:
  source:
    api_url: "https://github.source.int/api/v3"
    git_url: "https://github.source.int"
    app_id: 12345
    installation_id: 1111
    private_key_path: "/keys/source_app_private_key.pem"
    org: "org1"

  target:
    api_url: "https://github.target.int/api/v3"
    git_url: "https://github.target.int"
    app_id: 23456
    installation_id: 2222
    private_key_path: "/keys/target_app_private_key.pem"
    org: "org2"

repos:
  - "fred"
  - "barney"
  - "wilma"
  - "betty"

work_dir: "/tmp/repos"
```

The `work_dir` path tells the application where to create temporary bare clones during mirroring.
In a container this is usually inside the container filesystem.

## Building and running locally

Install Go version 1.24 or newer.

Fetch dependencies and run tests.

```bash
go test ./...
```

Build the binary.

```bash
go build ./cmd/gh-sync
```

Run the synchroniser with a config file.

```bash
./gh-sync -c config.yaml
```

You can also use the provided Makefile.

```bash
make build
make test
make coverage
```

## Docker usage

Build the image.

```bash
make docker-build
```

Run the image with a mounted configuration and keys directory.

```bash
make docker-run
```

This assumes you have `config.yaml` in the project root and a `keys` directory that contains the GitHub App private keys, as referenced in the configuration.

## Docker compose example

A simple compose file is included.

```bash
docker compose up --build
```

This builds the image if needed and runs a one shot sync job with the provided configuration.

## Makefile targets

Some useful commands.

```bash
make build          # build the gh-sync binary
make test           # run go test ./...
make coverage       # run tests with coverage
make watch          # watch for changes and rerun tests, requires entr
make dev            # run gh-sync locally with the default config path
make docker-build   # build the Docker image
make docker-run     # run the image with mounted config and keys
make publish        # tag and push the image to a registry
```

The `watch` target assumes the `entr` tool is available on your system.
On many Unix like systems you can install it with your package manager.

## Notes

The application shells out to the `git` command for clone and push operations.
Ensure that `git` is available in the runtime environment, both locally and in the container image.
