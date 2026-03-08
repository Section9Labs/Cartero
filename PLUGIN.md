# Plugins

Cartero discovers local plugin manifests from the `plugins/` directory.

The current refactor keeps plugins metadata-only and safe by default. A plugin manifest describes an integration point without granting Cartero permission to execute remote actions.

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
```

## Conformance workflow

1. add a workspace fixture under `internal/plugin/conformance/testdata/workspaces/<plugin-name>/plugins/`
2. capture the expected plain-text operator output in `plugin-list.plain.golden`
3. register the fixture in `internal/plugin/conformance/harness_test.go`
4. run `go test ./...` to execute manifest validation, fixture loading, and golden output checks

This workflow runs in CI through the normal Go test step, so new plugins inherit the same contract without extra pipeline wiring.
