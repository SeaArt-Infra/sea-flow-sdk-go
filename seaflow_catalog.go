package seaflowsdk

import "context"

// TemplateCatalogResource is the published template catalog. Its requests carry
// the caller's Production Key because OpenResty protects the /flow route.
type TemplateCatalogResource struct {
	transport *Transport
}

// Search queries the published catalog.
func (r *TemplateCatalogResource) Search(ctx context.Context, request CatalogSearchRequest) (*CatalogSearchResult, error) {
	var result CatalogSearchResult
	if err := r.transport.PostJSON(ctx, "/seaflow/template-catalog/search", request, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ReadGraph reads one published template's graph.
func (r *TemplateCatalogResource) ReadGraph(ctx context.Context, templateID string) (*TemplateGraphDetail, error) {
	if err := requireIdentifier("templateID", templateID); err != nil {
		return nil, err
	}
	var result TemplateGraphDetail
	if err := r.transport.PostJSON(ctx, "/seaflow/template-catalog/graph", map[string]string{"id": templateID}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
