package seaflowsdk

import (
	"net/http"
	"strings"
)

// ClientOptions configures a Client. The Production Key stays on the server
// side of whatever product embeds this SDK (ADR-0015: no package offers a
// browser-side mode).
type ClientOptions struct {
	// BaseURL is the gateway or Engine root, for example
	// "https://seainfra.dev/flow" or an in-cluster Service address. The SDK
	// derives APIBaseURL by adding /api/v1.
	BaseURL string
	// APIBaseURL overrides the derived API URL. It is useful for test servers and
	// deployments that mount the API below a non-standard path.
	APIBaseURL string
	// ProductionKey is the one Production Key bound to this SDK caller. It is
	// sent as Authorization: Bearer <token> on every request.
	ProductionKey string
	// EndUserID is the default end-user identifier sent as
	// `X-Infra-User-Id`. The Engine stores it as an opaque identifier and never
	// resolves it to an account (ADR-0010). Override per request with
	// WithEndUser.
	EndUserID string
	// Headers are extra headers sent on every request.
	Headers map[string]string
	// HTTPClient overrides the default client, which times out after 60 seconds.
	HTTPClient *http.Client
}

// Client is the whole Engine surface, grouped by resource.
type Client struct {
	BaseURL       string
	APIBaseURL    string
	ProductionKey string
	EndUserID     string
	Transport     *Transport

	// TemplateCatalog reads the published catalog through the same authenticated
	// gateway path as every other resource.
	TemplateCatalog *TemplateCatalogResource
	Templates       *TemplatesResource
	Workspaces      *WorkspacesResource
	Workflows       *WorkflowsResource
	Runs            *RunsResource
	Models          *ModelsResource
	Assets          *AssetsResource
	Records         *RecordsResource
}

// NewClient builds a client. A missing BaseURL or ProductionKey is reported by
// the first call that needs it, so a client can be constructed before its
// configuration is fully known.
func NewClient(options ClientOptions) *Client {
	transport := NewTransport(options)
	client := &Client{
		BaseURL:       transport.baseURL,
		APIBaseURL:    transport.apiBaseURL,
		ProductionKey: transport.productionKey,
		EndUserID:     transport.endUserID,
		Transport:     transport,
	}
	client.bindResources()
	return client
}

// WithEndUser copies the client so its requests name a different end user. The
// receiver is unchanged, and the copy shares the HTTP client and configuration —
// only the `X-Infra-User-Id` value differs. Use it when one process serves many
// end users, which is the normal shape for an integrating product's server.
func (c *Client) WithEndUser(endUserID string) *Client {
	if c == nil {
		return nil
	}
	clone := *c
	clone.EndUserID = strings.TrimSpace(endUserID)
	clone.Transport = c.Transport.WithEndUser(endUserID)
	clone.bindResources()
	return &clone
}

func (c *Client) bindResources() {
	c.TemplateCatalog = &TemplateCatalogResource{transport: c.Transport}
	c.Templates = &TemplatesResource{transport: c.Transport}
	c.Workspaces = &WorkspacesResource{transport: c.Transport}
	c.Workflows = &WorkflowsResource{transport: c.Transport}
	c.Runs = &RunsResource{transport: c.Transport}
	c.Models = &ModelsResource{transport: c.Transport}
	c.Assets = &AssetsResource{transport: c.Transport}
	c.Records = &RecordsResource{transport: c.Transport}
}
