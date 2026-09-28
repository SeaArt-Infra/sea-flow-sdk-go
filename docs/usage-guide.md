# SeaFlow Go SDK Usage Guide

The SDK exposes one `seaflowsdk.Client` with resource fields for the
project-facing SeaFlow API. It uses the standard library only and keeps the
Production Key in the server-side process that owns the integration.

## Configure the client

```go
client := seaflowsdk.NewClient(seaflowsdk.ClientOptions{
	BaseURL:       "https://seainfra.dev/flow",
	ProductionKey: os.Getenv("SEA_FLOW_SDK_PRODUCTION_KEY"),
	EndUserID:     "customer-42",
})
```

`BaseURL` is the OpenResty or Engine root; the SDK derives `/api/v1`. Set
`APIBaseURL` only for a non-standard mount or an isolated test server.
`ProductionKey` is the single project credential and is sent on every request as
`Authorization: Bearer <key>`. `EndUserID` becomes `X-Infra-User-Id`; it is an
opaque identifier derived from the integrating product's authenticated user,
not a second credential.

`WithEndUser` returns an independent client view that shares the HTTP client:

```go
user := client.WithEndUser("customer-42")
```

## Create, publish, and run a workflow

```go
ctx := context.Background()
workspace, err := user.Workspaces.Create(ctx, seaflowsdk.CreateWorkspaceRequest{Name: "My workflows"})
if err != nil { log.Fatal(err) }
workflow, err := user.Workspaces.CreateWorkflow(ctx, workspace.ID)
if err != nil { log.Fatal(err) }

graph := seaflowsdk.Graph{
	Nodes: []seaflowsdk.GraphNode{
		{ID: "input-1", Type: "workflow", Data: map[string]any{
			"kind": seaflowsdk.NodeKindInputText, "text": "一只小猫",
		}},
		{ID: "image-1", Type: "workflow", Data: map[string]any{
			"kind": seaflowsdk.NodeKindGenerateImage,
			"model": "nano_banana_2", "prompt": "一只小猫的插画",
		}},
	},
	Edges: []seaflowsdk.GraphEdge{{
		ID: "edge-1", Source: "input-1", Target: "image-1",
		SourceHandle: seaflowsdk.HandleText, TargetHandle: seaflowsdk.HandleText,
	}},
}
if _, err := user.Workflows.Save(ctx, workflow.ID,
	seaflowsdk.SaveWorkflowRequest{Name: "Poster", Graph: &graph}); err != nil {
	log.Fatal(err)
}
published, err := user.Workflows.Publish(ctx, workflow.ID)
if err != nil { log.Fatal(err) }
run, err := user.Workflows.CreateRun(ctx, workflow.ID, seaflowsdk.CreateRunRequest{})
if err != nil { log.Fatal(err) }
detail, err := user.Runs.Get(ctx, run.ID)
if err != nil { log.Fatal(err) }
fmt.Println(published.PublishedVersionID, detail.Run.Status)
```

`Save` replaces both the name and graph. Read the draft first when changing only
one field. A runtime-input graph must pass values in `CreateRunRequest`.

## Resources

- `TemplateCatalog.Search` and `ReadGraph` query the published catalog.
- `Templates.List`, `Publish`, `Get`, `Delete`, `Copy`, and `PublishVersion`
  manage the caller's catalog entries.
- `Workspaces.List`, `Create`, `Get`, `Rename`, `Delete`, `ListWorkflows`, and
  `CreateWorkflow` manage canvas containers.
- `Workflows.List`, `Create`, `Get`, `Save`, `Delete`, `Publish`, `PinCover`,
  `ListRuns`, and `CreateRun` manage canvases and execution snapshots.
- `Runs.Get` and `Stop` read or stop an execution.
- `Models.List` returns the live model catalog.
- `Assets.List` and `Records.List` read the end user's outputs and replayable
  history.

The Production Key authenticates the project and `EndUserID` scopes private
workspaces, draft canvases, assets, records, and runs inside it. The same
end-user ID in another project is isolated. Published templates are a shared
catalog: callers may list, read, and copy them, while only their creator may
manage an entry. Resources created without `EndUserID` remain project-shared;
SeaFlow cannot safely infer their historical end user. Omit `EndUserID` only
for a shared project workspace; it is required to start a project-scoped run.
Do not send `production_provider`; SeaFlow resolves it from the onboarded
project bound to the key.

## Graphs and identifiers

`GraphNode.Data` is intentionally `map[string]any`; it preserves canvas layout
fields such as `dimensions` and `handleBounds` during read-modify-write cycles.
Use the exported `NodeKind*`, `Handle*`, status, source, and content-type
constants instead of string literals. Path identifiers are escaped as one
segment by the SDK.

## Errors

HTTP/API failures are returned as `*APIError`; use `IsNotFound` and
`IsConflict` for common cases. Configuration mistakes are sentinels such as
`ErrMissingBaseURL`, `ErrMissingProductionKey`, and `ErrMissingIdentifier`.
Transport failures remain the underlying `net/http` error, so inspect them with
`errors.As` rather than API helpers.

## Testing

```bash
go test ./...
SEA_FLOW_SDK_BASE_URL=http://127.0.0.1:18081 \
SEA_FLOW_SDK_PRODUCTION_KEY=<project-key> \
go test -run TestEngineEndToEnd -v ./...
```

The end-to-end test writes to the configured Engine. Use a disposable instance;
it covers catalog access, workspace/canvas lifecycle, publish, list, typed
errors, records, assets, and request-shape handling. A real run requires a live
model gateway and is intentionally not part of the default SDK E2E test.
