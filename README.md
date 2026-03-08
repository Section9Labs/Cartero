# Cartero

Cartero is a modernized Go CLI for planning and validating security awareness exercises.
This refactor replaces the abandoned scaffold with:

- a polished terminal experience powered by Cobra and Lip Gloss
- safe-by-default campaign validation and preview commands
- schema-validated plugin discovery for local exercise integrations
- unit and CLI tests with a smoke-test workflow
- reproducible packaging with GoReleaser, Docker, and GitHub Actions

## Why this refactor

The previous repository state was not operational:

- the module was missing
- the entrypoint imported a non-existent package
- there was no usable command surface
- testing and release automation did not exist

This version focuses on a maintainable foundation that can evolve without reintroducing unsafe or brittle behavior.

## Quick start

```bash
make bootstrap
make build
./dist/cartero init
./dist/cartero preview --plain -f campaign.yaml
./dist/cartero validate --plain -f campaign.yaml
./dist/cartero doctor --plain
```

Run from a nested directory or an external shell session with `--root /path/to/workspace` to force workspace resolution.

## Commands

```text
cartero init         Write a starter campaign file
cartero preview      Render a styled readiness overview
cartero validate     Lint a campaign definition
cartero doctor       Inspect local project health
cartero plugin list  Show installed plugin manifests
cartero version      Print build metadata
```

## Campaign model

Campaigns are YAML files that describe an awareness exercise plan. Cartero validates structure and enforces local safety controls:

- `capture_credentials` must stay `false`
- `allow_external_links` must stay `false`
- manager approval is required for exercises marked `high` risk

Start from [`configs/campaign.example.yaml`](configs/campaign.example.yaml).

## Development

```bash
make fmt
make vet
make test
make smoke
```

CI runs on push and pull request via [`ci.yml`](.github/workflows/ci.yml). Tagged releases are packaged by [`release.yml`](.github/workflows/release.yml) using [`.goreleaser.yaml`](.goreleaser.yaml).
