package seaflowsdk

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

// This test drives a real Workflow Engine process. It is skipped unless both
// variables point at one, so the default `go test ./...` stays offline:
//
//	SEA_FLOW_SDK_BASE_URL=http://127.0.0.1:18081 \
//	SEA_FLOW_SDK_PRODUCTION_KEY=<the project's Production Key> \
//	go test -run TestEngineEndToEnd -v ./...
//
// Point it at a disposable engine. It writes workspaces, workflows and catalog
// state, so never aim it at a shared deployment.
func TestEngineEndToEnd(t *testing.T) {
	baseURL := strings.TrimSpace(os.Getenv("SEA_FLOW_SDK_BASE_URL"))
	if baseURL == "" {
		t.Skip("set SEA_FLOW_SDK_BASE_URL and SEA_FLOW_SDK_PRODUCTION_KEY to run the engine end-to-end test")
	}
	token := strings.TrimSpace(os.Getenv("SEA_FLOW_SDK_PRODUCTION_KEY"))
	if token == "" {
		t.Skip("set SEA_FLOW_SDK_BASE_URL and SEA_FLOW_SDK_PRODUCTION_KEY to run the engine end-to-end test")
	}
	endUser := strings.TrimSpace(os.Getenv("SEA_FLOW_SDK_END_USER"))
	if endUser == "" {
		endUser = "sdk-e2e-end-user"
	}
	ctx := context.Background()

	client := NewClient(ClientOptions{BaseURL: baseURL, ProductionKey: token, EndUserID: endUser})

	// 1. Catalog requests go through the same authenticated gateway path.
	catalog, err := client.TemplateCatalog.Search(ctx, CatalogSearchRequest{Limit: 5})
	if err != nil {
		t.Fatalf("catalog search: %v", err)
	}
	t.Logf("published catalog search returned %d templates", len(catalog.Templates))

	// 2. A credential the Engine cannot resolve is refused, not silently accepted.
	unresolved := NewClient(ClientOptions{BaseURL: baseURL, ProductionKey: "not-an-onboarded-token", EndUserID: endUser})
	if _, err := unresolved.Workspaces.List(ctx, ListOptions{}); err == nil {
		t.Fatal("an unonboarded token was accepted")
	} else {
		t.Logf("unonboarded token refused with: %v", err)
	}

	// 3. Create a workspace, then a canvas inside it.
	workspace, err := client.Workspaces.Create(ctx, CreateWorkspaceRequest{Name: "SDK 端到端测试"})
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	t.Cleanup(func() {
		if err := client.Workspaces.Delete(context.Background(), workspace.ID); err != nil && !IsNotFound(err) {
			t.Logf("cleanup: delete workspace %s: %v", workspace.ID, err)
		}
	})
	if workspace.ID == "" || workspace.Name != "SDK 端到端测试" {
		t.Fatalf("created workspace = %+v", workspace)
	}

	created, err := client.Workspaces.CreateWorkflow(ctx, workspace.ID)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	if created.ID == "" || created.WorkspaceID != workspace.ID {
		t.Fatalf("created workflow = %+v", created)
	}
	t.Cleanup(func() {
		// The test deletes this itself; a second delete is a no-op 404.
		if err := client.Workflows.Delete(context.Background(), created.ID); err != nil && !IsNotFound(err) {
			t.Logf("cleanup: delete workflow %s: %v", created.ID, err)
		}
	})

	// 4. Save a graph and read back what was written, layout fields included.
	// A text input feeding one generation node: publishing requires a generation
	// node (the Engine refuses a graph that produces nothing). The model name is
	// only read at run creation, which this test does not reach.
	graph := Graph{
		Nodes: []GraphNode{
			{
				ID:       "local-input-1",
				Type:     "workflow",
				Position: &NodePosition{X: 183, Y: 75},
				Data: map[string]any{
					"kind":         NodeKindInputText,
					"label":        "Text input",
					"text":         "一只小猫",
					"runtimeInput": true,
					"dimensions":   map[string]any{"width": 296, "height": 256},
				},
			},
			{
				ID:       "local-image-1",
				Type:     "workflow",
				Position: &NodePosition{X: 584, Y: 50},
				Data: map[string]any{
					"kind":   NodeKindGenerateImage,
					"label":  "Image generation",
					"model":  "nano_banana_2",
					"prompt": "一只小猫的插画",
					"params": map[string]any{"resolution": "1K", "aspect_ratio": "1:1"},
				},
			},
		},
		Edges: []GraphEdge{{
			ID:           "edge-1",
			Source:       "local-input-1",
			Target:       "local-image-1",
			SourceHandle: HandleText,
			TargetHandle: HandleText,
		}},
		Viewport: &Viewport{Zoom: 1},
	}
	saved, err := client.Workflows.Save(ctx, created.ID, SaveWorkflowRequest{Name: "SDK 端到端测试画布", Graph: &graph})
	if err != nil {
		t.Fatalf("save workflow: %v", err)
	}
	if saved.Name != "SDK 端到端测试画布" {
		t.Errorf("saved name = %q", saved.Name)
	}
	if len(saved.Graph.Nodes) != 2 {
		t.Fatalf("saved graph nodes = %+v", saved.Graph.Nodes)
	}
	if len(saved.Graph.Edges) != 1 {
		t.Fatalf("saved graph edges = %+v", saved.Graph.Edges)
	}
	if saved.Graph.Nodes[0].Data["text"] != "一只小猫" {
		t.Errorf("saved node text = %v", saved.Graph.Nodes[0].Data["text"])
	}
	if _, ok := saved.Graph.Nodes[0].Data["dimensions"]; !ok {
		t.Errorf("the canvas layout field did not survive the round trip: %v", saved.Graph.Nodes[0].Data)
	}
	if saved.Graph.Nodes[1].Data["model"] != "nano_banana_2" {
		t.Errorf("saved generation node model = %v", saved.Graph.Nodes[1].Data["model"])
	}

	// 5. Publish it, which mints the immutable version a run executes (ADR-0007).
	published, err := client.Workflows.Publish(ctx, created.ID)
	if err != nil {
		t.Fatalf("publish workflow: %v", err)
	}
	if published.PublishedVersionID == "" {
		t.Errorf("published workflow = %+v, want a publishedVersionId", published)
	}
	t.Logf("published version %s", published.PublishedVersionID)

	// 6. Reads that must agree with the writes.
	workflows, err := client.Workflows.List(ctx, ListOptions{Limit: 100})
	if err != nil {
		t.Fatalf("list workflows: %v", err)
	}
	if !containsWorkflow(workflows, created.ID) {
		t.Errorf("the created workflow is not in the list of %d", len(workflows))
	}
	inWorkspace, err := client.Workspaces.ListWorkflows(ctx, workspace.ID, ListOptions{Limit: 100})
	if err != nil {
		t.Fatalf("list workspace workflows: %v", err)
	}
	if !containsWorkflow(inWorkspace, created.ID) {
		t.Errorf("the created workflow is not in its workspace")
	}
	reloaded, err := client.Workspaces.Get(ctx, workspace.ID)
	if err != nil {
		t.Fatalf("get workspace: %v", err)
	}
	if reloaded.WorkflowCount < 1 {
		t.Errorf("workspace workflowCount = %d", reloaded.WorkflowCount)
	}
	runs, err := client.Workflows.ListRuns(ctx, created.ID, ListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if len(runs) != 0 {
		t.Errorf("a run exists for a workflow that was never run: %+v", runs)
	}
	if _, err := client.Records.List(ctx, ListOptions{Limit: 5}); err != nil {
		t.Fatalf("list records: %v", err)
	}
	if _, err := client.Assets.List(ctx, ListAssetsOptions{Limit: 5}); err != nil {
		t.Fatalf("list assets: %v", err)
	}

	// 7. A missing resource is a typed 404, not a generic failure.
	if _, err := client.Workflows.Get(ctx, "wf-does-not-exist"); !IsNotFound(err) {
		t.Errorf("Get of a missing workflow: err = %v, want a 404", err)
	}
	if _, err := client.Runs.Get(ctx, "run-does-not-exist"); !IsNotFound(err) {
		t.Errorf("Get of a missing run: err = %v, want a 404", err)
	}
	if _, err := client.Runs.Stop(ctx, "run-does-not-exist"); !IsNotFound(err) {
		t.Errorf("Stop of a missing run: err = %v, want a 404", err)
	}

	// 8. Delete, then confirm the Engine agrees it is gone.
	if err := client.Workflows.Delete(ctx, created.ID); err != nil {
		t.Fatalf("delete workflow: %v", err)
	}
	if _, err := client.Workflows.Get(ctx, created.ID); !IsNotFound(err) {
		t.Errorf("Get after delete: err = %v, want a 404", err)
	}

	// 9. Run a published graph through the live model gateway and wait for the
	// asynchronous node result before the workspace cleanup runs.
	var runGraph Graph
	rawGraph, err := json.Marshal(graph)
	if err != nil {
		t.Fatalf("clone graph: %v", err)
	}
	if err := json.Unmarshal(rawGraph, &runGraph); err != nil {
		t.Fatalf("clone graph: %v", err)
	}
	delete(runGraph.Nodes[0].Data, "runtimeInput")
	runner, err := client.Workspaces.CreateWorkflow(ctx, workspace.ID)
	if err != nil {
		t.Fatalf("create runner workflow: %v", err)
	}
	t.Cleanup(func() {
		if err := client.Workflows.Delete(context.Background(), runner.ID); err != nil && !IsNotFound(err) {
			t.Logf("cleanup: delete runner workflow %s: %v", runner.ID, err)
		}
	})
	if _, err := client.Workflows.Save(ctx, runner.ID, SaveWorkflowRequest{Name: "SDK 端到端运行测试", Graph: &runGraph}); err != nil {
		t.Fatalf("save runner workflow: %v", err)
	}
	if _, err := client.Workflows.Publish(ctx, runner.ID); err != nil {
		t.Fatalf("publish runner workflow: %v", err)
	}
	started, err := client.Workflows.CreateRun(ctx, runner.ID, CreateRunRequest{})
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	detail, err := waitForRun(client, started.ID)
	if err != nil {
		t.Fatalf("wait for run: %v", err)
	}
	if detail.Run.Status != RunStatusCompleted {
		t.Fatalf("run status = %q, want completed", detail.Run.Status)
	}
	if len(detail.NodeRuns) != 1 || detail.NodeRuns[0].Status != NodeRunStatusCompleted {
		t.Fatalf("node runs = %+v, want one completed node", detail.NodeRuns)
	}
	if len(detail.NodeRuns[0].Output) == 0 {
		t.Fatal("completed generation node has no output")
	}
	if _, err := client.Records.List(ctx, ListOptions{Limit: 20}); err != nil {
		t.Fatalf("list run records: %v", err)
	}
	if _, err := client.Assets.List(ctx, ListAssetsOptions{WorkflowID: runner.ID, Limit: 10}); err != nil {
		t.Fatalf("list run assets: %v", err)
	}
}

func waitForRun(client *Client, runID string) (*RunDetail, error) {
	deadline := time.Now().Add(3 * time.Minute)
	for time.Now().Before(deadline) {
		detail, err := client.Runs.Get(context.Background(), runID)
		if err != nil {
			return nil, err
		}
		switch detail.Run.Status {
		case RunStatusCompleted, RunStatusPartiallyFailed, RunStatusFailed, RunStatusStopped:
			return detail, nil
		}
		time.Sleep(2 * time.Second)
	}
	return nil, &APIError{Message: "run did not reach a terminal state within 3m"}
}

func containsWorkflow(workflows []Workflow, id string) bool {
	for _, workflow := range workflows {
		if workflow.ID == id {
			return true
		}
	}
	return false
}
