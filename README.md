# plugin-openapi-ai-context

Kalo plugin that reads an OpenAPI 3.1 specification and emits a condensed API surface summary (`api_contracts.yaml`) designed for LLM agent consumption.

## What it generates

Given an OpenAPI spec, the plugin produces a single YAML file that summarises every endpoint, grouped by tag:

```yaml
base_path: /api/v1
auth: Bearer JWT
endpoints:
  users:
    - method: GET
      path: /users
      response: User[]
      filters: [q, role]
    - method: POST
      path: /users
      body: UserCreate
      response: User
```

### Extracted fields

| Field | Source |
|-------|--------|
| `base_path` | `servers[0].url` |
| `auth` | Security scheme (bearer format or API key location) |
| `method` | HTTP method |
| `path` | Path template |
| `auth: false` | Operations that override global security with `security: []` |
| `body` | Request body `$ref` schema name |
| `response` | Success response `$ref` schema name (with `[]` suffix for arrays) |
| `filters` | Query parameter names |

## Input / Output

| | Format | Description |
|---|--------|-------------|
| **Input** | `KA:OA1:YAML1` | OpenAPI 3.1 YAML specification |
| **Output** | `KA:AC1:YAML1` | AI context file for LLM agents |

## Configuration

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `specFileName` | string | `openapi.yaml` | Name of the OpenAPI spec file to read |

## Pipeline example

```yaml
stores:
  openapi-spec:
    type: filesystem
    path: docs/openapi

  ai-context:
    type: filesystem
    path: ai-context

pipelines:
  compile:
    stages:
      - name: "ai-context-api"
        steps:
          - plugin: "@kalo-build/plugin-openapi-ai-context"
            inputStore: openapi-spec
            outputStore: ai-context
            config:
              specFileName: openapi.yaml
```

## Project layout

```
cmd/plugin/         CLI entry point (JSON config → generate)
pkg/generate/       Core logic (parse, convert, emit)
testdata/
  input/            Sample OpenAPI spec for tests
  ground-truth/     Expected output for compile test
plugin.yaml         Kalo plugin manifest
```

## Build

```bash
go build -o plugin-openapi-ai-context ./cmd/plugin
```

WASM (for Kalo CLI):

```bash
GOOS=wasip1 GOARCH=wasm go build -o dist/plugin.wasm ./cmd/plugin
```

## Test

```bash
go test ./... -v
```
