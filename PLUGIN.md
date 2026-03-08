# Plugins

Cartero discovers local plugin manifests from the `plugins/` directory.

The current refactor keeps plugins metadata-only and safe by default. A plugin manifest describes an integration point without granting Cartero permission to execute remote actions.

## Manifest schema

```yaml
name: local-preview
version: 1.0.0
kind: renderer
mode: local-only
safe: true
description: Writes previews into a local review sink.
```

## Goals

- make available integrations visible to operators
- keep the CLI offline-first and easy to audit
- support future extension points without coupling the command layer to one transport

## Usage

```bash
cartero plugin list
```
