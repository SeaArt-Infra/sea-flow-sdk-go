---
name: sea-flow-sdk-go
description: Implement or troubleshoot server-side SeaFlow workflow integrations with the Go SDK. Use for template, workspace, workflow, run, model, asset, or record operations; do not use for browser clients.
---

# Sea Flow SDK Go

Use `github.com/SeaArt-Infra/sea-flow-sdk-go` with Go 1.24.3+.

## Scope

- Use this SDK from server code only. Keep `ProductionKey` in the host
  application's existing secret mechanism; do not expose it to a browser.
- Use the SDK rather than duplicating its REST transport or adding a
  `production_provider` request field.
- Read `docs/usage-guide.md` before using a resource method not covered here.

## Client and identity

```go
import (
	"os"

	seaflowsdk "github.com/SeaArt-Infra/sea-flow-sdk-go"
)

client := seaflowsdk.NewClient(seaflowsdk.ClientOptions{
	BaseURL:       os.Getenv("SEA_FLOW_BASE_URL"),
	ProductionKey: os.Getenv("SEA_FLOW_SDK_PRODUCTION_KEY"),
})
user := client.WithEndUser(currentUserID)
```

- Construct one shared client per server configuration. `WithEndUser` returns a
  copy with a different `X-Infra-User-Id`, which scopes that user's private
  workspaces, draft workflows, assets, records, and runs within the project.
- Published templates are shared catalog entries. Any caller may read or copy
  one, while only its creator may manage it.
- Derive the end-user ID from the integrating product's authenticated user. Do
  not send it as a model-selected argument or expose the Production Key to a browser.

## Workflow changes

- Standard write flow: create or copy a workspace workflow, `Save` its complete
  graph and name, `Publish`, then `CreateRun`.
- `Workflows.Save` replaces both name and graph. For a partial change, read the
  workflow first and preserve the field that is not changing.
- Treat `GraphNode.Data` as opaque JSON. Preserve canvas layout and unknown node
  fields during read-modify-write operations.
- `CreateRun` schedules asynchronous work. A concurrent active run can return a
  conflict; do not retry it blindly.

## Errors

- `ErrMissingBaseURL`, `ErrMissingProductionKey`, and `ErrMissingIdentifier`
  are local errors before a request. Use `errors.Is` to distinguish them.
- API failures are `*APIError`. Use `IsNotFound` and `IsConflict` for 404 and
  409 handling; use `errors.As` when the caller needs the error details.
