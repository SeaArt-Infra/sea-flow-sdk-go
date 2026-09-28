package seaflowsdk

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type capturedRequest struct {
	Method   string
	Path     string
	RawQuery string
	Body     string
}

// captureServer records the requests a test makes and answers each with data.
func captureServer(t *testing.T, data string) (*httptest.Server, *[]capturedRequest) {
	t.Helper()
	requests := &[]capturedRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		*requests = append(*requests, capturedRequest{
			Method:   r.Method,
			Path:     r.URL.Path,
			RawQuery: r.URL.RawQuery,
			Body:     string(raw),
		})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":` + data + `}`))
	}))
	t.Cleanup(server.Close)
	return server, requests
}

func newTestClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	return NewClient(ClientOptions{BaseURL: baseURL, ProductionKey: "sdk-test-token", EndUserID: "end-user-1"})
}

// TestEveryResourceUsesItsContractPath pins each method to the path and verb in
// docs/seaflow-openapi.yaml. A typo here is a 404 the Engine would only
// report at run time.
func TestEveryResourceUsesItsContractPath(t *testing.T) {
	cases := []struct {
		name       string
		wantMethod string
		wantPath   string
		wantQuery  string
		wantBody   string
		call       func(ctx context.Context, client *Client) error
	}{
		{
			name: "catalog search", wantMethod: http.MethodPost, wantPath: "/api/v1/seaflow/template-catalog/search",
			wantBody: `{"query":"海报","limit":5}`,
			call: func(ctx context.Context, c *Client) error {
				_, err := c.TemplateCatalog.Search(ctx, CatalogSearchRequest{Query: "海报", Limit: 5})
				return err
			},
		},
		{
			name: "catalog graph", wantMethod: http.MethodPost, wantPath: "/api/v1/seaflow/template-catalog/graph",
			wantBody: `{"id":"tpl-1"}`,
			call: func(ctx context.Context, c *Client) error {
				_, err := c.TemplateCatalog.ReadGraph(ctx, "tpl-1")
				return err
			},
		},
		{
			name: "list templates", wantMethod: http.MethodGet, wantPath: "/api/v1/seaflow/templates", wantQuery: "limit=10",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Templates.List(ctx, ListOptions{Limit: 10})
				return err
			},
		},
		{
			name: "publish template", wantMethod: http.MethodPost, wantPath: "/api/v1/seaflow/templates",
			wantBody: `{"workflowId":"wf-1","name":"海报","category":"marketing","tags":["a","b"]}`,
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Templates.Publish(ctx, TemplateSaveRequest{
					WorkflowID: "wf-1", Name: "海报", Category: "marketing", Tags: []string{"a", "b"},
				})
				return err
			},
		},
		{
			name: "get template", wantMethod: http.MethodGet, wantPath: "/api/v1/seaflow/templates/tpl-1",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Templates.Get(ctx, "tpl-1")
				return err
			},
		},
		{
			name: "delete template", wantMethod: http.MethodDelete, wantPath: "/api/v1/seaflow/templates/tpl-1",
			call: func(ctx context.Context, c *Client) error { return c.Templates.Delete(ctx, "tpl-1") },
		},
		{
			name: "copy template", wantMethod: http.MethodPost, wantPath: "/api/v1/seaflow/templates/tpl-1/copies",
			wantBody: `{"workspaceId":"ws-1","runtimeInputs":{"story":"一只小猫"}}`,
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Templates.Copy(ctx, "tpl-1", CopyTemplateRequest{
					WorkspaceID: "ws-1", RuntimeInputs: map[string]any{"story": "一只小猫"},
				})
				return err
			},
		},
		{
			name: "publish template version", wantMethod: http.MethodPost, wantPath: "/api/v1/seaflow/templates/tpl-1/versions",
			wantBody: `{"workflowId":"wf-1","name":"海报 v2"}`,
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Templates.PublishVersion(ctx, "tpl-1", TemplateSaveRequest{WorkflowID: "wf-1", Name: "海报 v2"})
				return err
			},
		},
		{
			name: "list workspaces", wantMethod: http.MethodGet, wantPath: "/api/v1/seaflow/workspaces",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Workspaces.List(ctx, ListOptions{})
				return err
			},
		},
		{
			name: "create workspace", wantMethod: http.MethodPost, wantPath: "/api/v1/seaflow/workspaces", wantBody: `{"name":"我的工作区"}`,
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Workspaces.Create(ctx, CreateWorkspaceRequest{Name: "我的工作区"})
				return err
			},
		},
		{
			name: "get workspace", wantMethod: http.MethodGet, wantPath: "/api/v1/seaflow/workspaces/ws-1",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Workspaces.Get(ctx, "ws-1")
				return err
			},
		},
		{
			name: "rename workspace", wantMethod: http.MethodPatch, wantPath: "/api/v1/seaflow/workspaces/ws-1", wantBody: `{"name":"新名字"}`,
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Workspaces.Rename(ctx, "ws-1", RenameWorkspaceRequest{Name: "新名字"})
				return err
			},
		},
		{
			name: "delete workspace", wantMethod: http.MethodDelete, wantPath: "/api/v1/seaflow/workspaces/ws-1",
			call: func(ctx context.Context, c *Client) error { return c.Workspaces.Delete(ctx, "ws-1") },
		},
		{
			name: "list workspace workflows", wantMethod: http.MethodGet, wantPath: "/api/v1/seaflow/workspaces/ws-1/workflows", wantQuery: "limit=5",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Workspaces.ListWorkflows(ctx, "ws-1", ListOptions{Limit: 5})
				return err
			},
		},
		{
			name: "create workspace workflow", wantMethod: http.MethodPost, wantPath: "/api/v1/seaflow/workspaces/ws-1/workflows",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Workspaces.CreateWorkflow(ctx, "ws-1")
				return err
			},
		},
		{
			name: "list workflows", wantMethod: http.MethodGet, wantPath: "/api/v1/seaflow/workflows",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Workflows.List(ctx, ListOptions{})
				return err
			},
		},
		{
			name: "create workflow", wantMethod: http.MethodPost, wantPath: "/api/v1/seaflow/workflows",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Workflows.Create(ctx)
				return err
			},
		},
		{
			name: "get workflow", wantMethod: http.MethodGet, wantPath: "/api/v1/seaflow/workflows/wf-1",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Workflows.Get(ctx, "wf-1")
				return err
			},
		},
		{
			name: "save workflow", wantMethod: http.MethodPatch, wantPath: "/api/v1/seaflow/workflows/wf-1",
			wantBody: `{"name":"海报","graph":{"nodes":[{"id":"n1","data":{"kind":"input-text","text":"一只小猫"}}]}}`,
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Workflows.Save(ctx, "wf-1", SaveWorkflowRequest{
					Name: "海报",
					Graph: &Graph{Nodes: []GraphNode{{
						ID: "n1",
						Data: map[string]any{
							"kind": NodeKindInputText,
							"text": "一只小猫",
						},
					}}},
				})
				return err
			},
		},
		{
			name: "delete workflow", wantMethod: http.MethodDelete, wantPath: "/api/v1/seaflow/workflows/wf-1",
			call: func(ctx context.Context, c *Client) error { return c.Workflows.Delete(ctx, "wf-1") },
		},
		{
			name: "publish workflow", wantMethod: http.MethodPost, wantPath: "/api/v1/seaflow/workflows/wf-1/publish",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Workflows.Publish(ctx, "wf-1")
				return err
			},
		},
		{
			name: "pin cover", wantMethod: http.MethodPut, wantPath: "/api/v1/seaflow/workflows/wf-1/cover", wantBody: `{"nodeRunId":"nr-1"}`,
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Workflows.PinCover(ctx, "wf-1", PinCoverRequest{NodeRunID: "nr-1"})
				return err
			},
		},
		{
			name: "list runs", wantMethod: http.MethodGet, wantPath: "/api/v1/seaflow/workflows/wf-1/runs", wantQuery: "offset=10",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Workflows.ListRuns(ctx, "wf-1", ListOptions{Offset: 10})
				return err
			},
		},
		{
			name: "create run", wantMethod: http.MethodPost, wantPath: "/api/v1/seaflow/workflows/wf-1/runs",
			wantBody: `{"runtimeInputs":{"story":"一只小猫"}}`,
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Workflows.CreateRun(ctx, "wf-1", CreateRunRequest{RuntimeInputs: map[string]any{"story": "一只小猫"}})
				return err
			},
		},
		{
			name: "get run", wantMethod: http.MethodGet, wantPath: "/api/v1/seaflow/runs/run-1",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Runs.Get(ctx, "run-1")
				return err
			},
		},
		{
			name: "stop run", wantMethod: http.MethodPatch, wantPath: "/api/v1/seaflow/runs/run-1",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Runs.Stop(ctx, "run-1")
				return err
			},
		},
		{
			name: "list models", wantMethod: http.MethodGet, wantPath: "/api/v1/seaflow/models",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Models.List(ctx)
				return err
			},
		},
		{
			name: "list assets", wantMethod: http.MethodGet, wantPath: "/api/v1/seaflow/assets", wantQuery: "limit=3&workflowId=wf-1",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Assets.List(ctx, ListAssetsOptions{WorkflowID: "wf-1", Limit: 3})
				return err
			},
		},
		{
			name: "list records", wantMethod: http.MethodGet, wantPath: "/api/v1/seaflow/records",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Records.List(ctx, ListOptions{})
				return err
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			server, requests := captureServer(t, `null`)
			client := newTestClient(t, server.URL)
			if err := testCase.call(context.Background(), client); err != nil {
				t.Fatalf("call: %v", err)
			}
			if len(*requests) != 1 {
				t.Fatalf("saw %d requests, want 1", len(*requests))
			}
			seen := (*requests)[0]
			if seen.Method != testCase.wantMethod {
				t.Errorf("method = %s, want %s", seen.Method, testCase.wantMethod)
			}
			if seen.Path != testCase.wantPath {
				t.Errorf("path = %s, want %s", seen.Path, testCase.wantPath)
			}
			if seen.RawQuery != testCase.wantQuery {
				t.Errorf("query = %q, want %q", seen.RawQuery, testCase.wantQuery)
			}
			if testCase.wantBody != "" && seen.Body != testCase.wantBody {
				t.Errorf("body = %s, want %s", seen.Body, testCase.wantBody)
			}
		})
	}
}

func TestIdentifiersAreEscapedIntoSinglePathSegments(t *testing.T) {
	server, requests := captureServer(t, `null`)
	client := newTestClient(t, server.URL)

	if _, err := client.Workflows.Get(context.Background(), "wf 1/../../etc"); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(*requests) != 1 {
		t.Fatalf("saw %d requests, want 1", len(*requests))
	}

	// The property that matters is that the identifier can add no path segment:
	// it reaches the Engine escaped, so the router still sees one.
	seen := (*requests)[0].Path
	const prefix = "/api/v1/seaflow/workflows/"
	if !strings.HasPrefix(seen, prefix) {
		t.Fatalf("path = %q, want the %q prefix", seen, prefix)
	}
	if remainder := strings.TrimPrefix(seen, prefix); strings.Contains(remainder, "/") {
		t.Errorf("path = %q, want the identifier kept inside one segment", seen)
	}
}

// A caller mistake is refused locally and reported as ErrMissingIdentifier, the
// same kind of error as a missing baseURL or token, so errors.Is tells it from
// the Engine's own answer.
func TestMissingIdentifierFailsBeforeTheRequest(t *testing.T) {
	server, requests := captureServer(t, `null`)
	client := newTestClient(t, server.URL)

	cases := []struct {
		name string
		// want names the field the message has to point the caller at, so the
		// refusal is actionable and not just "something was missing".
		want string
		call func() error
	}{
		{
			name: "Get with a blank workflow id",
			want: "workflowID",
			call: func() error {
				_, err := client.Workflows.Get(context.Background(), "  ")
				return err
			},
		},
		{
			name: "Stop with a blank run id",
			want: "runID",
			call: func() error {
				_, err := client.Runs.Stop(context.Background(), "")
				return err
			},
		},
		{
			name: "CreateWorkspace with a blank name",
			want: "request.Name",
			call: func() error {
				_, err := client.Workspaces.Create(context.Background(), CreateWorkspaceRequest{})
				return err
			},
		},
		{
			// The Engine writes the name it is given, so a save without one would
			// blank the workflow's name instead of leaving it alone.
			name: "Save with a blank name",
			want: "request.Name",
			call: func() error {
				_, err := client.Workflows.Save(context.Background(), "wf-1", SaveWorkflowRequest{Graph: &Graph{}})
				return err
			},
		},
		{
			name: "Save with a whitespace name",
			want: "request.Name",
			call: func() error {
				_, err := client.Workflows.Save(context.Background(), "wf-1", SaveWorkflowRequest{Name: "   "})
				return err
			},
		},
		{
			name: "Copy with no destination workspace",
			want: "request.WorkspaceID",
			call: func() error {
				_, err := client.Templates.Copy(context.Background(), "tpl-1", CopyTemplateRequest{})
				return err
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.call()
			if err == nil {
				t.Fatal("err = nil, want a refusal")
			}
			if !errors.Is(err, ErrMissingIdentifier) {
				t.Fatalf("err = %v, want ErrMissingIdentifier", err)
			}
			if !strings.Contains(err.Error(), testCase.want+" is required") {
				t.Errorf("err = %v, want it to name %q", err, testCase.want)
			}
		})
	}
	if len(*requests) != 0 {
		t.Errorf("saw %d requests, want none", len(*requests))
	}
}

// A request's unset fields are left out of the body rather than sent as Go zero
// values, and a route the contract declares without a requestBody is sent with
// no body at all. The Engine writes what it is given, so an empty string or a
// zero is not the same as "not mentioned".
func TestUnsetRequestFieldsAreOmittedFromTheBody(t *testing.T) {
	server, requests := captureServer(t, `null`)
	client := newTestClient(t, server.URL)

	if _, err := client.Workflows.Save(context.Background(), "wf-1", SaveWorkflowRequest{Name: "海报"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// An empty search sends no fields at all: the Engine treats a missing
	// category and limit as its own defaults, where "" and 0 are values it reads.
	if _, err := client.TemplateCatalog.Search(context.Background(), CatalogSearchRequest{}); err != nil {
		t.Fatalf("Search with nothing: %v", err)
	}
	if _, err := client.TemplateCatalog.Search(context.Background(), CatalogSearchRequest{Query: "poster"}); err != nil {
		t.Fatalf("Search: %v", err)
	}
	if _, err := client.Workspaces.CreateWorkflow(context.Background(), "ws-1"); err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	// create-run is the one input route the canvas also calls with nothing, and
	// the Engine decodes its body, so an empty request is still sent as {}.
	if _, err := client.Workflows.CreateRun(context.Background(), "wf-1", CreateRunRequest{}); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}

	want := []string{`{"name":"海报"}`, `{}`, `{"query":"poster"}`, ``, `{}`}
	if len(*requests) != len(want) {
		t.Fatalf("saw %d requests, want %d", len(*requests), len(want))
	}
	for index, body := range want {
		if seen := (*requests)[index].Body; seen != body {
			t.Errorf("request %d body = %q, want %q", index, seen, body)
		}
	}
}

// TestGraphRoundTripKeepsCanvasLayoutFields is the reason GraphNode.Data is a
// map: a graph read from the Engine and written back must not lose the canvas's
// own layout fields.
func TestGraphRoundTripKeepsCanvasLayoutFields(t *testing.T) {
	raw := `{
		"nodes": [{
			"id": "local-1",
			"type": "workflow",
			"position": {"x": 183, "y": 75},
			"data": {
				"kind": "input-text",
				"label": "Text input",
				"runtimeInput": true,
				"text": "一只小猫",
				"handleBounds": {"source": [{"id": "text"}]},
				"dimensions": {"width": 296, "height": 256}
			}
		}],
		"edges": [{
			"id": "e1", "source": "local-1", "target": "local-2",
			"sourceHandle": "text", "targetHandle": "text"
		}],
		"viewport": {"x": 0, "y": 0, "zoom": 1}
	}`

	var graph Graph
	if err := json.Unmarshal([]byte(raw), &graph); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(graph.Nodes) != 1 || graph.Nodes[0].Data["kind"] != NodeKindInputText {
		t.Fatalf("nodes = %+v", graph.Nodes)
	}

	encoded, err := json.Marshal(graph)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, key := range []string{"handleBounds", "dimensions", "runtimeInput", "text"} {
		if !strings.Contains(string(encoded), key) {
			t.Errorf("re-encoded graph dropped %q: %s", key, encoded)
		}
	}
}

func TestRunDetailUnwrapsTheRunAndItsNodeRuns(t *testing.T) {
	server, _ := captureServer(t, `{
		"run": {"id":"run-1","workflowId":"wf-1","status":"completed","stopRequested":false},
		"nodeRuns": [{
			"id":"nr-1","runId":"run-1","nodeId":"local-1","model":"nano_banana_2",
			"status":"completed","httpStatus":200,"progress":1,
			"output":[{"content":[{"type":"image","url":"https://cdn/1.webp"}]}]
		}]
	}`)
	client := newTestClient(t, server.URL)

	detail, err := client.Runs.Get(context.Background(), "run-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if detail.Run.ID != "run-1" || detail.Run.Status != RunStatusCompleted {
		t.Errorf("run = %+v", detail.Run)
	}
	if len(detail.NodeRuns) != 1 {
		t.Fatalf("nodeRuns = %+v", detail.NodeRuns)
	}
	node := detail.NodeRuns[0]
	if node.Status != NodeRunStatusCompleted || node.HTTPStatus != http.StatusOK {
		t.Errorf("node = %+v", node)
	}
	if len(node.Output) != 1 || len(node.Output[0].Content) != 1 || node.Output[0].Content[0].URL != "https://cdn/1.webp" {
		t.Errorf("node output = %+v", node.Output)
	}
}

func TestTemplateDetailUnwrapsTemplateAndVersion(t *testing.T) {
	server, _ := captureServer(t, `{
		"id":"tpl-1","slug":"poster","name":"海报","status":"published","latestVersion":2,"isOwner":true,
		"steps":[{"kind":"generate-image","label":"Image generation","model":"nano_banana_2"}],
		"version":{"id":"tv-2","templateId":"tpl-1","version":2,"status":"published","assets":["https://cdn/1.webp"]}
	}`)
	client := newTestClient(t, server.URL)

	detail, err := client.Templates.Get(context.Background(), "tpl-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if detail.ID != "tpl-1" || detail.Slug != "poster" || detail.LatestVersion != 2 {
		t.Errorf("template = %+v", detail.Template)
	}
	if len(detail.Steps) != 1 || detail.Steps[0].Model != "nano_banana_2" {
		t.Errorf("steps = %+v", detail.Steps)
	}
	if detail.Version == nil || detail.Version.Version != 2 {
		t.Errorf("version = %+v", detail.Version)
	}
}

func TestAssetListNarrowsToAWorkspace(t *testing.T) {
	server, requests := captureServer(t, `[{"id":"asset-1","runId":"run-1","model":"nano_banana_2"}]`)
	client := newTestClient(t, server.URL)

	assets, err := client.Assets.List(context.Background(), ListAssetsOptions{WorkspaceID: "ws-1"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(assets) != 1 || assets[0].ID != "asset-1" {
		t.Errorf("assets = %+v", assets)
	}
	if seen := (*requests)[0].RawQuery; seen != "workspaceId=ws-1" {
		t.Errorf("query = %q, want only the set filter", seen)
	}
}

func TestWithEndUserKeepsEveryResourceBound(t *testing.T) {
	server, requests := captureServer(t, `null`)
	client := newTestClient(t, server.URL).WithEndUser("user-a")

	if _, err := client.Workspaces.List(context.Background(), ListOptions{}); err != nil {
		t.Fatalf("List: %v", err)
	}
	if _, err := client.Runs.Stop(context.Background(), "run-1"); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if len(*requests) != 2 {
		t.Fatalf("saw %d requests, want 2", len(*requests))
	}
}
