---
name: sea-flow-sdk-go
description: Build and troubleshoot SeaFlow integrations with the Go SDK. Use when creating, publishing, copying, or running workflows, browsing templates and models, or diagnosing Production Key and end-user attribution errors.
---

# SeaFlow Go SDK

Use `github.com/SeaArt-Infra/sea-flow-sdk-go` with Go 1.24.3+.

## Install

```bash
go get github.com/SeaArt-Infra/sea-flow-sdk-go
```

## Workflow

1. Create one `seaflowsdk.Client` with the OpenResty root and the onboarded
   Production Key; reuse it.
2. Use `WithEndUser` for the current product user. It sends `X-Infra-User-Id`;
   it does not change project ownership.
3. Create or copy a workflow, save its graph, publish it, then create a run.
4. Read run details and assets; stop a run with `Runs.Stop` when needed.
5. Use `TemplateCatalog` for published templates and `Models` for the live
   model list. Never add a `production_provider` request field.

## Initialize

```go
client := seaflowsdk.NewClient(seaflowsdk.ClientOptions{
	BaseURL: "https://seainfra.dev/flow", ProductionKey: os.Getenv("SEA_FLOW_SDK_PRODUCTION_KEY"),
})
user := client.WithEndUser("customer-42")
```

Every request carries `Authorization: Bearer <ProductionKey>`. The key stays on
the server side; this SDK has no browser mode.

## Core call

```go
workspace, _ := user.Workspaces.Create(ctx, seaflowsdk.CreateWorkspaceRequest{Name: "Demo"})
workflow, _ := user.Workspaces.CreateWorkflow(ctx, workspace.ID)
_, _ = user.Workflows.Save(ctx, workflow.ID, seaflowsdk.SaveWorkflowRequest{Name: "Demo", Graph: &graph})
_, _ = user.Workflows.Publish(ctx, workflow.ID)
run, _ := user.Workflows.CreateRun(ctx, workflow.ID, seaflowsdk.CreateRunRequest{})
detail, _ := user.Runs.Get(ctx, run.ID)
```

## Errors

Handle `*APIError` with `IsNotFound`/`IsConflict`. Handle
`ErrMissingBaseURL`, `ErrMissingProductionKey`, and `ErrMissingIdentifier`
before treating a failure as a server response.

## Route reference

- `TemplateCatalog.Search`, `ReadGraph`
- `Templates.List`, `Publish`, `Get`, `Delete`, `Copy`, `PublishVersion`
- `Workspaces.List`, `Create`, `Get`, `Rename`, `Delete`, `ListWorkflows`, `CreateWorkflow`
- `Workflows.List`, `Create`, `Get`, `Save`, `Delete`, `Publish`, `PinCover`, `ListRuns`, `CreateRun`
- `Runs.Get`, `Stop`; `Models.List`; `Assets.List`; `Records.List`
