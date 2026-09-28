# sea-flow-sdk-go

> First release. The contract is maintained in SeaFlow at `docs/seaflow-openapi.yaml`.
> in the Workflow Engine repository; the Engine never depends on this package.

Go SDK for the Workflow Engine (SeaFlow). It wraps the project-facing API for
templates, workspaces, workflows, versions, runs and assets, so an integrating
product can publish a canvas and run it by id without hand-rolling HTTP.

## Available Resources

| Resource | Client field | What it does |
| --- | --- | --- |
| Template catalog | `client.TemplateCatalog` | Search the published catalog and read a template's graph with the caller's Production Key |
| Templates | `client.Templates` | List, publish, read, take down, copy, and version catalog entries |
| Workspaces | `client.Workspaces` | Create, list, rename, delete a workspace; create and list its canvases |
| Workflows | `client.Workflows` | Create a draft, save its graph, publish it, pin a cover, start a run |
| Runs | `client.Runs` | Read a run with its node records, and stop it |
| Models | `client.Models` | List the models the canvas may bind to execution nodes |
| Assets | `client.Assets` | List the terminal outputs of the caller's runs |
| Records | `client.Records` | List the caller's replayable run history |

## How It Works

1. Create a `Client` with the OpenResty root URL and the caller's Production Key.
2. Every request carries that same key as `Authorization: Bearer <token>`. OpenResty
   authenticates it, then SeaFlow resolves the workflow project bound to it.
3. The end user travels per request as `X-Infra-User-Id`. Set a default with
   `ClientOptions.EndUserID`, or copy the client per request with `WithEndUser`.
4. The Engine answers with the platform envelope `{"code":0,"message":"ok","data":...}`.
   The SDK unwraps `data` into typed values and turns a failure into `*APIError`.

## Quick Start

```bash
go get github.com/SeaArt-Infra/sea-flow-sdk-go
```

Requires Go 1.24.3 or newer, matching the version declared in `go.mod`.

```go
package main

import (
	"context"
	"fmt"
	"os"

	seaflowsdk "github.com/SeaArt-Infra/sea-flow-sdk-go"
)

func main() {
	ctx := context.Background()
	client := seaflowsdk.NewClient(seaflowsdk.ClientOptions{
		BaseURL:       "https://seainfra.dev/flow",
		ProductionKey: os.Getenv("SEA_FLOW_SDK_PRODUCTION_KEY"),
	})

	// The same client serves many end users; only the identifier changes.
	user := client.WithEndUser("customer-42")

	workspace, err := user.Workspaces.Create(ctx, seaflowsdk.CreateWorkspaceRequest{Name: "My workflows"})
	if err != nil {
		panic(err)
	}
	workflow, err := user.Workspaces.CreateWorkflow(ctx, workspace.ID)
	if err != nil {
		panic(err)
	}

	graph := seaflowsdk.Graph{
		Nodes: []seaflowsdk.GraphNode{
			{
				ID:   "input-1",
				Type: "workflow",
				Data: map[string]any{
					"kind": seaflowsdk.NodeKindInputText,
					"text": "一只小猫",
				},
			},
			{
				ID:   "image-1",
				Type: "workflow",
				Data: map[string]any{
					"kind":   seaflowsdk.NodeKindGenerateImage,
					"model":  "nano_banana_2",
					"prompt": "一只小猫的插画",
				},
			},
		},
		Edges: []seaflowsdk.GraphEdge{{
			ID:           "edge-1",
			Source:       "input-1",
			Target:       "image-1",
			SourceHandle: seaflowsdk.HandleText,
			TargetHandle: seaflowsdk.HandleText,
		}},
	}
	if _, err := user.Workflows.Save(ctx, workflow.ID, seaflowsdk.SaveWorkflowRequest{
		Name:  "Poster",
		Graph: &graph,
	}); err != nil {
		panic(err)
	}
	published, err := user.Workflows.Publish(ctx, workflow.ID)
	if err != nil {
		panic(err)
	}
	fmt.Println("published version:", published.PublishedVersionID)

	// Running it by id is the whole point of the service.
	run, err := user.Workflows.CreateRun(ctx, workflow.ID, seaflowsdk.CreateRunRequest{})
	if err != nil {
		panic(err)
	}
	detail, err := user.Runs.Get(ctx, run.ID)
	if err != nil {
		panic(err)
	}
	fmt.Println("run status:", detail.Run.Status)
}
```

## Configuration

| `ClientOptions` field | Required | Meaning |
| --- | --- | --- |
| `BaseURL` | yes, unless `APIBaseURL` is set | Gateway or Engine root. The SDK derives `<BaseURL>/api/v1`; use `https://seainfra.dev/flow` for OpenResty |
| `APIBaseURL` | no | Explicit full API prefix, mainly for test servers or non-standard mounts |
| `ProductionKey` | yes | The one Production Key bound to this caller. Every API request carries it as Bearer authentication |
| `EndUserID` | no | Default `X-Infra-User-Id`. The Engine stores it as an opaque identifier and never resolves it to an account |
| `Headers` | no | Extra headers on every request. `Authorization` is always replaced with `ProductionKey`; set `EndUserID` rather than an `X-Infra-User-Id` header |
| `HTTPClient` | no | Overrides the default client, which times out after 60 seconds |

`Client.WithEndUser(id)` returns a copy with a different end-user identifier. The
receiver is unchanged and the copy shares the HTTP client, so one process can serve
many end users with one connection pool.

## Ownership

Ownership follows the Production Key's bound project, not the end user:

- `owner` is the bound project, so **workspaces, canvases, assets and
  records are project-scoped** — every end user of that product shares them.
- Runs are separated per end user: `Workflows.ListRuns` and `Runs.Get` only ever answer
  with the caller's own runs.

The catalog still carries the same key even though SeaFlow itself does not use it for
catalog ownership: OpenResty requires it before forwarding `/flow` requests.

## Errors

Every failure the Engine answers with is an `*APIError` carrying the HTTP status, the
body `code` and the message:

```go
_, err := client.Workflows.Get(ctx, "wf-missing")
if seaflowsdk.IsNotFound(err) {
	// 404: the workflow does not exist, or is not visible to this caller
}
if seaflowsdk.IsConflict(err) {
	// 409: an active run blocks the action, or a duplicate exists
}
```

A non-zero body `code` is treated as a failure even when the HTTP status is 2xx. A
non-2xx status whose body is not JSON also produces an `*APIError`, with the body quoted
back as its message.

A `2xx` whose body is not the Engine's envelope is *not* an `*APIError`: there is no
failure envelope to carry, so the rejection is an error naming the request and wrapping
the decode failure. That covers an HTML error page from a proxy that answered 200.

Three failures are raised before any request is sent, because they are caller mistakes
rather than server answers, and all three are sentinels for `errors.Is`:
`ErrMissingBaseURL`, `ErrMissingProductionKey`, and `ErrMissingIdentifier` for a blank
path identifier or a required request field (`Workflows.Get(ctx, "")`,
`Workflows.Save(ctx, id, SaveWorkflowRequest{Name: name})`).

A failure below HTTP is not wrapped in an `*APIError` either: a refused connection or an
expired timeout returns the `*url.Error` from `net/http`, so test it with `errors.As`
rather than with `IsNotFound`, which is only about a 404.

## Saving a draft

`Workflows.Save(ctx, id, SaveWorkflowRequest{Name, Graph})` replaces both fields: the
Engine writes the name it is given and validates the graph it is given, so a request that
omits the name blanks the name rather than leaving it alone. When only one of the two is
changing, read the draft with `Workflows.Get` first and send both back.

The SDK refuses a save without a name (`ErrMissingIdentifier`), so a title cannot
disappear by accident. A save without a graph reaches the Engine and is refused there
with a 400 (`graph must be valid JSON`).

## Identifiers and graphs

Every path identifier is escaped into a single path segment, so an identifier can never
add a route segment.

`GraphNode.Data` is a `map[string]any` on purpose: the canvas stores its own layout fields
(`dimensions`, `handleBounds`, …) next to the documented ones, and a struct would drop
them on a read-modify-write round trip. The documented keys are `kind` (one of the
`NodeKind*` constants, required), `label`, `model`, `prompt`, `params`, `text`, `assetUrl`
and `runtimeInput`. Edge handles are the `Handle*` constants, and both ends of an edge must
carry the same medium.

## Testing

```bash
go test ./...                 # offline: request shapes, envelope handling, error typing
```

The end-to-end test drives a real Engine process and is skipped unless you point it at
one. Use a disposable engine — it writes workspaces, canvases and catalog state:

```bash
SEA_FLOW_SDK_BASE_URL=http://127.0.0.1:18081 SEA_FLOW_SDK_PRODUCTION_KEY=<the Production Key> go test -run TestEngineEndToEnd -v ./...
```

It covers: authenticated catalog access, an unbound Production Key being refused, create
workspace → create canvas → save a graph → read it back → publish → list workflows,
workspace workflows, workspace counts, runs, records and assets → typed 404s → delete.

Not covered end to end: starting a run, because that needs a live model gateway and a
credential that may call models. `CreateRun`, `Runs.Get` and `Runs.Stop` are covered by
the offline tests for request shape and error typing; the live path is exercised by the
Engine's own tests. There is no browser-side mode: the credential stays on the server.

See [docs/usage-guide.md](docs/usage-guide.md) for the complete developer guide and
[skills/sea-flow-sdk-go/SKILL.md](skills/sea-flow-sdk-go/SKILL.md) for the Agent skill.

<script type="text/plain" data-doc-skill data-doc-skill-id="sea-flow-sdk-go" data-doc-skill-label="Sea Flow SDK Go" data-doc-skill-filename="sea-flow-sdk-go-SKILL.md" data-doc-skill-version="1">
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
  copy with a different `X-Infra-User-Id`; it does not change project ownership.
- Workspaces, workflows, templates, assets, and records belong to the project
  bound to the credential. End-user identity scopes run activity, not ownership.

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
</script>
