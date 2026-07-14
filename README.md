# GitHub Enterprise Repository Synchroniser in Go

This application synchronises repositories between two GitHub Organisations/Enterprise Server instances. It uses GitHub Apps for authentication on both the source instance and the target instance. For each configured repository pair it performs a bare clone from the source repository, then a mirror push into the target repository.

The code is structured as a Go module with an internal package layout.
A Makefile is provided for common tasks, and Docker and compose files let you run sync jobs in a container.

> A helm deployment project is in the works to allow this project to be deployed as a cronjob to allow the application to be deployed to Kubernetes to run as a scheduled task.

## Prerequisites

Before using this application, ensure the following requirements are met:

1. **Repositories must exist**: Both source and target repositories must already be created in their respective GitHub Enterprise organisations. The application does not create repositories.

2. **GitHub Apps configured**: You need two GitHub Apps set up:
   - One GitHub App installed on the source GitHub Organisation/Enterprise instance
   - One GitHub App installed on the target GitHub Organisation/Enterprise instance
   - See [Creating a GitHub App](https://docs.github.com/en/apps/creating-github-apps/registering-a-github-app/registering-a-github-app) for setup instructions

3. **GitHub App permissions**: Each GitHub App must have the following permissions:
   - **Repository permissions**:
     - Contents: Read (source) / Read & Write (target)
     - Metadata: Read
   - The apps must be installed in the respective organisations with access to the repositories you want to synchronise
   - See [Setting permissions for GitHub Apps](https://docs.github.com/en/apps/creating-github-apps/registering-a-github-app/choosing-permissions-for-a-github-app) for more details

4. **Private keys**: You must have the private key files (`.pem`) for both GitHub Apps accessible to the application.
   - See [Generating a private key](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/managing-private-keys-for-github-apps) for instructions on creating and downloading private keys

5. **Git installed**: The `git` command-line tool must be available in the runtime environment.

## Configuration

The application expects a `config.yaml` file with this structure.

```yaml
github:
  source:
    api_url: "https://github.source.int/api/v3"
    git_url: "https://github.source.int"
    app_id: 12345
    installation_id: 1111
    private_key_path: "keys/source_app_private_key.pem"
    org: "org1"

  target:
    api_url: "https://github.target.int/api/v3"
    git_url: "https://github.target.int"
    app_id: 23456
    installation_id: 2222
    private_key_path: "keys/target_app_private_key.pem"
    org: "org2"

repos:
  - source: "fred"
    target: "barney"
  - source: "wilma"
    target: "betty"
  - source: "pebbles"
    target: "pebbles"

work_dir: ".work/repos"
```

Each element in `repos` section defines a mapping from a source repository name to a target repository name.
The source repository is cloned from the source GitHub Enterprise organisation.
The same content is then mirrored into the target repository name in the target organisation.

The `work_dir` path tells the application where to create temporary bare clones during mirroring.

## Building and running locally

Install Go version 1.22 or newer.

Run tests.

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

Or use the Makefile.

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

This assumes you have `config.yaml` in the project root and a `keys` directory that contains the GitHub App private keys referenced in the configuration.

## Docker compose

A simple compose file is included so you can run a one shot sync.

```bash
docker compose up --build
```

## Makefile targets

Useful commands.

```bash
make build          # build the gh-sync binary
make test           # run go test ./...
make coverage       # run tests with coverage
make watch          # watch for file changes and rerun tests (requires entr)
make dev            # run gh-sync locally with the default config path
make docker-build   # build the Docker image
make docker-run     # run the image with mounted config and keys
```

> The `watch` target assumes the `entr` tool is available on your system.

## Notes

The application shells out to the `git` command for clone and push operations. Ensure that `git` is available in the runtime environment, both locally and in the container image.
