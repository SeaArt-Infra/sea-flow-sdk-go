package seaflowsdk

import "time"

// Node kinds a canvas node may carry in Graph.data.kind. The Engine validates
// both on save and on run creation.
const (
	NodeKindInputText     = "input-text"
	NodeKindInputImage    = "input-image"
	NodeKindInputVideo    = "input-video"
	NodeKindInputAudio    = "input-audio"
	NodeKindGenerateText  = "generate-text"
	NodeKindGenerateImage = "generate-image"
	NodeKindGenerateVideo = "generate-video"
	NodeKindGenerateAudio = "generate-audio"
)

// Media handles an edge connects on. GraphEdge.SourceHandle and TargetHandle
// are validated against these, and the two ends must carry the same medium.
const (
	HandleText  = "text"
	HandleImage = "image"
	HandleVideo = "video"
	HandleAudio = "audio"
)

// Model sources ModelsResource.List reports.
const (
	ModelSourceMultimodal = "multimodal"
	ModelSourceLLM        = "llm"
)

// Workflow.status values.
const (
	WorkflowStatusDraft  = "draft"
	WorkflowStatusActive = "active"
)

// Run.status values.
const (
	RunStatusQueued          = "queued"
	RunStatusRunning         = "running"
	RunStatusStopping        = "stopping"
	RunStatusStopped         = "stopped"
	RunStatusCompleted       = "completed"
	RunStatusPartiallyFailed = "partially_failed"
	RunStatusFailed          = "failed"
)

// NodeRun.status values.
const (
	NodeRunStatusPending     = "pending"
	NodeRunStatusDispatching = "dispatching"
	NodeRunStatusSubmitted   = "submitted"
	NodeRunStatusCompleted   = "completed"
	NodeRunStatusFailed      = "failed"
	NodeRunStatusSkipped     = "skipped"
)

// Template.status values.
const (
	TemplateStatusDraft     = "draft"
	TemplateStatusPublished = "published"
	TemplateStatusOffline   = "offline"
)

// ContentItem.type values.
const (
	ContentTypeText  = "text"
	ContentTypeImage = "image"
	ContentTypeVideo = "video"
	ContentTypeAudio = "audio"
	ContentTypeFile  = "file"
)

// ListOptions is the limit/offset pair every list route takes. Zero means the
// Engine's default.
type ListOptions struct {
	Limit  int
	Offset int
}

func (o ListOptions) query() QueryParams {
	return QueryParams{"limit": o.Limit, "offset": o.Offset}
}

// Graph is the canvas graph, in both directions: what SaveWorkflow sends and
// what GetWorkflow returns. Limits are enforced by the Engine (100 nodes, 200
// edges, 16KB per node payload, 8KB prompt, 16KB params).
type Graph struct {
	Nodes    []GraphNode `json:"nodes,omitempty"`
	Edges    []GraphEdge `json:"edges,omitempty"`
	Viewport *Viewport   `json:"viewport,omitempty"`
}

// GraphNode carries its payload in Data so that a graph read from the Engine can
// be written back unchanged. Data is a map rather than a struct because the
// canvas stores its own layout fields alongside the documented ones; a struct
// would silently drop them on the round trip.
//
// The documented keys are: kind (one of the NodeKind constants, required),
// label, model, prompt, params, text, assetUrl and runtimeInput.
type GraphNode struct {
	ID       string         `json:"id"`
	Type     string         `json:"type,omitempty"`
	Position *NodePosition  `json:"position,omitempty"`
	Data     map[string]any `json:"data,omitempty"`
}

// NodePosition is the node's canvas coordinate.
type NodePosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// GraphEdge connects two nodes on one medium.
type GraphEdge struct {
	ID           string `json:"id,omitempty"`
	Source       string `json:"source"`
	Target       string `json:"target"`
	SourceHandle string `json:"sourceHandle,omitempty"`
	TargetHandle string `json:"targetHandle,omitempty"`
}

// Viewport is the canvas scroll and zoom.
type Viewport struct {
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Zoom float64 `json:"zoom"`
}

// Workspace is the container that groups one end user's canvases.
type Workspace struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	WorkflowCount int       `json:"workflowCount"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// Workflow is a canvas: a draft the caller owns, or a published workflow shared
// with the caller's production line. IsOwner=false means read-only.
type Workflow struct {
	ID                 string    `json:"id"`
	WorkspaceID        string    `json:"workspaceId"`
	IsOwner            bool      `json:"isOwner"`
	Name               string    `json:"name"`
	Status             string    `json:"status"`
	Graph              Graph     `json:"graph"`
	TemplateID         string    `json:"templateId"`
	TemplateVersionID  string    `json:"templateVersionId"`
	TemplateAssets     []string  `json:"templateAssets"`
	PublishedVersionID string    `json:"publishedVersionId"`
	PreviewURL         string    `json:"previewUrl"`
	CoverURL           string    `json:"coverUrl"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

// Run is one execution of a workflow snapshot.
type Run struct {
	ID            string         `json:"id"`
	WorkflowID    string         `json:"workflowId"`
	VersionID     string         `json:"versionId"`
	Snapshot      Graph          `json:"snapshot"`
	RuntimeInputs map[string]any `json:"runtimeInputs"`
	Status        string         `json:"status"`
	StopRequested bool           `json:"stopRequested"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
}

// RunDetail is one run together with every node record it created.
type RunDetail struct {
	Run      Run       `json:"run"`
	NodeRuns []NodeRun `json:"nodeRuns"`
}

// NodeRun is one node's state inside a run.
type NodeRun struct {
	ID            string         `json:"id"`
	RunID         string         `json:"runId"`
	NodeID        string         `json:"nodeId"`
	Model         string         `json:"model"`
	GatewayTaskID string         `json:"gatewayTaskId"`
	Input         []ContentItem  `json:"input"`
	Params        map[string]any `json:"params"`
	Status        string         `json:"status"`
	Output        []NodeOutput   `json:"output"`
	Error         string         `json:"error"`
	HTTPStatus    int            `json:"httpStatus"`
	Progress      float64        `json:"progress"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
}

// NodeOutput is one entry of a node's output list. A generation node produces
// one output holding the generated content.
type NodeOutput struct {
	Content []ContentItem `json:"content"`
}

// ContentItem is one typed piece of content: text, or a URL to produced media.
type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	URL  string `json:"url,omitempty"`
}

// Asset is a terminal output of the caller's run, as the asset library lists it.
type Asset struct {
	ID              string       `json:"id"`
	RunID           string       `json:"runId"`
	WorkflowID      string       `json:"workflowId"`
	WorkflowName    string       `json:"workflowName"`
	WorkflowDeleted bool         `json:"workflowDeleted"`
	NodeID          string       `json:"nodeId"`
	NodeLabel       string       `json:"nodeLabel"`
	Model           string       `json:"model"`
	Output          []NodeOutput `json:"output"`
	CreatedAt       time.Time    `json:"createdAt"`
}

// RunRecord is one replayable run as the records list shows it: enough to draw
// the row, without the snapshot or the node records.
type RunRecord struct {
	ID            string    `json:"id"`
	WorkflowID    string    `json:"workflowId"`
	WorkflowName  string    `json:"workflowName"`
	WorkspaceID   string    `json:"workspaceId"`
	WorkspaceName string    `json:"workspaceName"`
	CoverURL      string    `json:"coverUrl"`
	CanvasDeleted bool      `json:"canvasDeleted"`
	Status        string    `json:"status"`
	NodeCount     int       `json:"nodeCount"`
	CreatedAt     time.Time `json:"createdAt"`
}

// Model is one model the canvas may bind to an execution node, as the gateway
// catalog reports it under the caller's entitlement.
type Model struct {
	Name             string   `json:"name"`
	DisplayName      string   `json:"displayName"`
	InputModalities  []string `json:"inputModalities"`
	OutputModalities []string `json:"outputModalities"`
	Source           string   `json:"source"`
}

// TemplateStep is one node summary shown on a template card.
type TemplateStep struct {
	Kind  string `json:"kind"`
	Label string `json:"label"`
	Model string `json:"model"`
}

// TemplateSummary is a published template as the catalog lists it.
type TemplateSummary struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Category    string         `json:"category"`
	Tags        []string       `json:"tags"`
	Steps       []TemplateStep `json:"steps"`
}

// TemplateGraphDetail is a catalog entry plus the graph its publisher shipped.
type TemplateGraphDetail struct {
	TemplateSummary
	Graph Graph `json:"graph"`
}

// Template is one catalog entry the caller can see.
type Template struct {
	ID            string         `json:"id"`
	Slug          string         `json:"slug"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	Category      string         `json:"category"`
	Tags          []string       `json:"tags"`
	Steps         []TemplateStep `json:"steps"`
	Status        string         `json:"status"`
	LatestVersion int            `json:"latestVersion"`
	PreviewURL    string         `json:"previewUrl"`
	CreatedBy     string         `json:"createdBy"`
	IsOwner       bool           `json:"isOwner"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
}

// TemplateVersion is one immutable published version of a template.
type TemplateVersion struct {
	ID          string    `json:"id"`
	TemplateID  string    `json:"templateId"`
	Version     int       `json:"version"`
	Status      string    `json:"status"`
	Graph       Graph     `json:"graph"`
	Assets      []string  `json:"assets"`
	PreviewURL  string    `json:"previewUrl"`
	CreatedAt   time.Time `json:"createdAt"`
	PublishedAt time.Time `json:"publishedAt"`
}

// TemplateDetail is a template with its current published version.
type TemplateDetail struct {
	Template
	Version *TemplateVersion `json:"version"`
}

// CatalogSearchRequest is the catalog search body.
type CatalogSearchRequest struct {
	Query    string `json:"query,omitempty"`
	Category string `json:"category,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

// CatalogSearchResult is the catalog search's data payload.
type CatalogSearchResult struct {
	Templates []TemplateSummary `json:"templates"`
}

// CreateWorkspaceRequest names a new workspace.
type CreateWorkspaceRequest struct {
	Name string `json:"name"`
}

// RenameWorkspaceRequest renames an existing workspace.
type RenameWorkspaceRequest struct {
	Name string `json:"name"`
}

// SaveWorkflowRequest saves a draft's name and graph. The Engine writes the
// name it is given, so an omitted Name blanks the name rather than leaving it
// alone; the SDK requires it for that reason. The graph is validated rather
// than merged, so send the one Workflows.Get returned: a request without it is
// refused with a 400.
type SaveWorkflowRequest struct {
	Name  string `json:"name,omitempty"`
	Graph *Graph `json:"graph,omitempty"`
}

// CreateRunRequest starts a run. RuntimeInputs holds the values for the nodes
// the author marked as runtime inputs; every one of them is required.
type CreateRunRequest struct {
	RuntimeInputs map[string]any `json:"runtimeInputs,omitempty"`
}

// PinCoverRequest pins a completed node run's image as the workflow's cover.
type PinCoverRequest struct {
	NodeRunID string `json:"nodeRunId"`
}

// CopyTemplateRequest copies a template into a workspace as a new workflow.
type CopyTemplateRequest struct {
	WorkspaceID   string         `json:"workspaceId,omitempty"`
	RuntimeInputs map[string]any `json:"runtimeInputs,omitempty"`
}

// TemplateSaveRequest publishes the caller's own workflow as a catalog entry.
// WorkflowID must be the caller's draft; a workflow merely visible through the
// production line cannot be published.
type TemplateSaveRequest struct {
	WorkflowID     string   `json:"workflowId"`
	Name           string   `json:"name"`
	Description    string   `json:"description,omitempty"`
	Category       string   `json:"category,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	CoverNodeRunID string   `json:"coverNodeRunId,omitempty"`
}

// ListAssetsOptions narrows the asset list to one workspace or one workflow.
type ListAssetsOptions struct {
	WorkspaceID string
	WorkflowID  string
	Limit       int
	Offset      int
}
