# Plugins

Cartero discovers local plugin manifests from the `plugins/` directory.

Plugin manifests describe trust boundaries and capabilities. First-party plugins in this repo also have corresponding Go command implementations that operate entirely on local workspace data by default.

## Manifest schema

```yaml
schema_version: v1
name: local-preview
version: 1.0.0
kind: renderer
mode: local-only
safe: true
capabilities:
  - preview.render
trust:
  level: first-party
  review_required: false
description: Writes previews into a local review sink.
```

## Required fields

- `schema_version`: currently must be `v1`
- `name`: lowercase plugin identifier using letters, digits, and hyphens
- `version`: semantic version such as `1.0.0`
- `kind`: `renderer`, `content-pack`, `importer`, `exporter`, or `integration`
- `mode`: `local-only`, `operator-review`, or `external-service`
- `safe`: whether the plugin stays within Cartero's safe local execution boundary
- `capabilities`: one or more declared extension points
- `trust.level`: `first-party`, `reviewed`, or `unreviewed`
- `trust.review_required`: must be `true` for unreviewed or external-service plugins

## Capability catalog

- `preview.render`: render local previews or review artifacts
- `campaign.template`: provide reusable campaign or landing-page content
- `campaign.import`: ingest source content into Cartero campaign drafts
- `audience.sync`: sync local or remote audience segment definitions
- `results.export`: write operator or executive reporting outputs
- `events.ingest`: collect reported-mailbox or training telemetry
- `webhook.deliver`: emit signed campaign lifecycle events

## Validation rules

- capabilities must be compatible with the declared `kind`
- `safe: true` is only valid for `local-only` and `operator-review` plugins
- `trust.review_required` must stay `true` when `safe: false`, `mode: external-service`, or `trust.level: unreviewed`
- invalid manifests are reported with field-level diagnostics during discovery and `doctor`

## Goals

- make available integrations visible to operators
- keep the CLI offline-first and easy to audit
- support future extension points without coupling the command layer to one transport

## Usage

```bash
cartero plugin list
cartero plugin sync
```

## Built-in plugins

- `local-preview`: renders local previews and review output
- `template-library`: seeds a curated local template pack into the embedded workspace store
- `clone-importer`: converts reviewed `.eml` or raw message files into safe Cartero drafts
- `analytics-export`: exports workspace analytics to JSON or CSV
- `audience-sync`: imports local CSV segments into the embedded workspace store
- `engagement-recorder`: stores non-sensitive engagement telemetry for reporting

## Embedded workspace state

First-party plugins use the embedded BoltDB workspace store at `.cartero/cartero.db`. This replaces the old MongoDB-dependent model with a portable local database that needs no separate service, credentials, or runtime management.

## Conformance workflow

1. add a workspace fixture under `internal/plugin/conformance/testdata/workspaces/<plugin-name>/plugins/`
2. capture the expected plain-text operator output in `plugin-list.plain.golden`
3. register the fixture in `internal/plugin/conformance/harness_test.go`
4. run `go test ./...` to execute manifest validation, fixture loading, and golden output checks

This workflow runs in CI through the normal Go test step, so new plugins inherit the same contract without extra pipeline wiring.
