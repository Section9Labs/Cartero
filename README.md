# Cartero

Cartero is now a Go CLI with an embedded, no-configuration workspace database instead of the old MongoDB-dependent runtime.

The current implementation provides:

- a polished terminal experience powered by Cobra and Lip Gloss
- a local admin web UI and safe testing pages via `cartero serve`
- safe-by-default campaign validation and preview commands
- an embedded SQLite workspace store at `.cartero/cartero.sqlite`
- first-party plugins for template seeding, clone import, audience sync, analytics export, and engagement recording
- normalized findings import from CSV, JSON, SARIF, and JSONL
- one-way legacy Mongo export migration into the current workspace
- unit, CLI, and conformance tests with a smoke-test workflow
- reproducible packaging with GoReleaser, Docker, and GitHub Actions

## Workspace model

Cartero keeps local state inside the active workspace:

- `.cartero/cartero.sqlite`: embedded SQLite state store
- `plugins/`: synced plugin manifests
- `drafts/`: generated campaign drafts from reviewed messages
- `exports/`: analytics exports

There is no external database to install or manage. Existing Bolt-backed workspaces are migrated into SQLite automatically on first open.

## Quick start

```bash
make bootstrap
make build

./dist/cartero --plain workspace init
./dist/cartero init campaign.yaml
./dist/cartero --plain validate -f campaign.yaml
./dist/cartero --plain preview -f campaign.yaml
./dist/cartero serve --addr 127.0.0.1:8080
./dist/cartero --plain template list
./dist/cartero --plain finding list
./dist/cartero --plain plugin list
```

Run from a nested directory or an external shell session with `--root /path/to/workspace` to force workspace resolution.

## Commands

```text
cartero workspace init       Bootstrap the embedded workspace
cartero workspace status     Show database and workspace counts
cartero init                 Write a starter campaign file
cartero preview              Render a styled readiness overview and persist a snapshot
cartero validate             Lint a campaign definition and persist a snapshot
cartero serve                Run the local admin UI and safe testing pages
cartero template list        Browse the seeded template library
cartero template show        Inspect a template in detail
cartero audience import      Import a CSV segment into the workspace
cartero audience list        List stored audience members
cartero import clone         Convert a reviewed message into a safe draft campaign
cartero finding import       Normalize external findings into the workspace
cartero finding list         List imported findings
cartero migrate mongo-export Import legacy Mongo export files into SQLite
cartero event record         Record engagement telemetry
cartero event list           List stored engagement events
cartero report export        Export workspace analytics to JSON or CSV
cartero plugin list          Show installed plugin manifests
cartero plugin sync          Sync built-in plugin manifests and template seed data
cartero doctor               Inspect local workspace health
cartero version              Print build metadata
```

## Built-in plugins

The repo ships with these first-party plugins:

- `local-preview`
- `template-library`
- `clone-importer`
- `analytics-export`
- `audience-sync`
- `engagement-recorder`

Their manifests live in [`plugins/`](plugins), and their contract is documented in [`PLUGIN.md`](PLUGIN.md).

## Campaign model

Campaigns are YAML files that describe an awareness exercise plan. Cartero validates structure and enforces local safety controls:

- `capture_credentials` must stay `false`
- `allow_external_links` must stay `false`
- manager approval is required for exercises marked `high` risk

Start from [`configs/campaign.example.yaml`](configs/campaign.example.yaml).

## Findings and migration

Cartero can correlate external scanner output with the same local workspace used for campaigns and events:

```bash
./dist/cartero --plain finding import --file scans/nuclei.jsonl --source nightly-nuclei
./dist/cartero --plain finding list --tool nuclei
./dist/cartero --plain migrate mongo-export --path legacy-export
```

The legacy migration path imports old Mongo export files for people and hits, and converts legacy credential artifacts into redacted findings instead of carrying raw submitted values forward.

## Development

```bash
make fmt
make vet
make test
make smoke
```

CI runs on push and pull request via [`ci.yml`](.github/workflows/ci.yml). Tagged releases are packaged by [`release.yml`](.github/workflows/release.yml) using [`.goreleaser.yaml`](.goreleaser.yaml).
