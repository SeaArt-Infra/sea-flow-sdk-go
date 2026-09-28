package seaflowsdk

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// recordingServer answers every request with one fixed status and body.
func recordingServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server
}

func TestCredentialedCallCarriesTokenAndEndUser(t *testing.T) {
	var seen *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":[]}`))
	}))
	t.Cleanup(server.Close)

	client := NewClient(ClientOptions{
		BaseURL:       server.URL,
		ProductionKey: "sdk-test-token",
		EndUserID:     "end-user-1",
	})
	if _, err := client.Workspaces.List(context.Background(), ListOptions{Limit: 20, Offset: 5}); err != nil {
		t.Fatalf("List: %v", err)
	}
	if seen == nil {
		t.Fatal("the server saw no request")
	}
	if seen.Method != http.MethodGet {
		t.Errorf("method = %s, want GET", seen.Method)
	}
	if seen.URL.Path != "/api/v1/seaflow/workspaces" {
		t.Errorf("path = %s", seen.URL.Path)
	}
	if got := seen.URL.Query().Get("limit"); got != "20" {
		t.Errorf("limit = %q, want 20", got)
	}
	if got := seen.URL.Query().Get("offset"); got != "5" {
		t.Errorf("offset = %q, want 5", got)
	}
	if got := seen.Header.Get("Authorization"); got != "Bearer sdk-test-token" {
		t.Errorf("Authorization = %q", got)
	}
	if got := seen.Header.Get(EndUserHeader); got != "end-user-1" {
		t.Errorf("%s = %q", EndUserHeader, got)
	}
}

func TestZeroPaginationValuesAreOmitted(t *testing.T) {
	var rawQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":[]}`))
	}))
	t.Cleanup(server.Close)

	client := NewClient(ClientOptions{BaseURL: server.URL, ProductionKey: "sdk-test-token"})
	if _, err := client.Workspaces.List(context.Background(), ListOptions{}); err != nil {
		t.Fatalf("List: %v", err)
	}
	if rawQuery != "" {
		t.Errorf("raw query = %q, want empty", rawQuery)
	}
}

func TestEndUserIsSetPerRequestWithoutTouchingTheOriginal(t *testing.T) {
	var headers []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers = append(headers, r.Header.Get(EndUserHeader))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":[]}`))
	}))
	t.Cleanup(server.Close)

	shared := NewClient(ClientOptions{BaseURL: server.URL, ProductionKey: "sdk-test-token", EndUserID: "default-user"})
	first := shared.WithEndUser("user-a")
	second := shared.WithEndUser("user-b")

	if _, err := first.Workspaces.List(context.Background(), ListOptions{}); err != nil {
		t.Fatalf("user-a List: %v", err)
	}
	if _, err := second.Workspaces.List(context.Background(), ListOptions{}); err != nil {
		t.Fatalf("user-b List: %v", err)
	}
	if _, err := shared.Workspaces.List(context.Background(), ListOptions{}); err != nil {
		t.Fatalf("shared List: %v", err)
	}

	want := []string{"user-a", "user-b", "default-user"}
	if len(headers) != len(want) {
		t.Fatalf("saw %d requests, want %d", len(headers), len(want))
	}
	for i, expected := range want {
		if headers[i] != expected {
			t.Errorf("request %d end user = %q, want %q", i, headers[i], expected)
		}
	}
	if shared.EndUserID != "default-user" {
		t.Errorf("the original client's end user changed to %q", shared.EndUserID)
	}
}

func TestCatalogRequestsCarryTheProductionKey(t *testing.T) {
	const body = `{"code":0,"message":"ok","data":{"templates":[{"id":"tpl-1","name":"海报"}]}}`

	var seen *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	client := NewClient(ClientOptions{
		BaseURL:       server.URL,
		ProductionKey: "sdk-test-token",
		EndUserID:     "end-user-1",
		Headers:       map[string]string{"Authorization": "Bearer wrong-key", "X-Extra": "kept"},
	})
	result, err := client.TemplateCatalog.Search(context.Background(), CatalogSearchRequest{Query: "海报", Limit: 5})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(result.Templates) != 1 || result.Templates[0].ID != "tpl-1" {
		t.Errorf("templates = %+v", result.Templates)
	}
	if got := seen.Header.Get("Authorization"); got != "Bearer sdk-test-token" {
		t.Errorf("Authorization = %q, want the configured ProductionKey", got)
	}
	if got := seen.Header.Get(EndUserHeader); got != "end-user-1" {
		t.Errorf("%s = %q, want the configured end user", EndUserHeader, got)
	}
	if got := seen.Header.Get("X-Extra"); got != "kept" {
		t.Errorf("X-Extra = %q, want the caller's other headers kept", got)
	}
}

func TestBaseURLDerivesTheFlowAPIPath(t *testing.T) {
	var seenPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":[]}`))
	}))
	t.Cleanup(server.Close)

	client := NewClient(ClientOptions{BaseURL: server.URL + "/flow", ProductionKey: "sdk-test-token"})
	if _, err := client.Models.List(context.Background()); err != nil {
		t.Fatalf("List: %v", err)
	}
	if seenPath != "/flow/api/v1/seaflow/models" {
		t.Errorf("path = %q, want /flow/api/v1/seaflow/models", seenPath)
	}
}

func TestAPIBaseURLOverridesTheDerivedPath(t *testing.T) {
	var seenPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":[]}`))
	}))
	t.Cleanup(server.Close)

	client := NewClient(ClientOptions{APIBaseURL: server.URL + "/custom", ProductionKey: "sdk-test-token"})
	if _, err := client.Models.List(context.Background()); err != nil {
		t.Fatalf("List: %v", err)
	}
	if seenPath != "/custom/seaflow/models" {
		t.Errorf("path = %q, want /custom/seaflow/models", seenPath)
	}
}

func TestCredentialedCallWithoutATokenFailsBeforeTheRequest(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	t.Cleanup(server.Close)

	client := NewClient(ClientOptions{BaseURL: server.URL})
	_, err := client.Workspaces.List(context.Background(), ListOptions{})
	if !errors.Is(err, ErrMissingProductionKey) {
		t.Fatalf("err = %v, want ErrMissingProductionKey", err)
	}
	if called {
		t.Error("the SDK sent a request it had already refused to sign")
	}
}

func TestMissingBaseURLFailsClearly(t *testing.T) {
	client := NewClient(ClientOptions{ProductionKey: "sdk-test-token"})
	_, err := client.Templates.List(context.Background(), ListOptions{})
	if !errors.Is(err, ErrMissingBaseURL) {
		t.Fatalf("err = %v, want ErrMissingBaseURL", err)
	}
}

func TestErrorEnvelopeBecomesAnAPIError(t *testing.T) {
	server := recordingServer(t, http.StatusNotFound, `{"code":404,"message":"workflow not found"}`)
	client := NewClient(ClientOptions{BaseURL: server.URL, ProductionKey: "sdk-test-token"})

	_, err := client.Workflows.Get(context.Background(), "wf-1")
	var apiError *APIError
	if !errors.As(err, &apiError) {
		t.Fatalf("err = %v, want *APIError", err)
	}
	if apiError.HTTPStatus != http.StatusNotFound || apiError.Code != 404 || apiError.Message != "workflow not found" {
		t.Errorf("APIError = %+v", apiError)
	}
	if !IsNotFound(err) {
		t.Error("IsNotFound(err) = false")
	}
	if IsConflict(err) {
		t.Error("IsConflict(err) = true")
	}
}

func TestConflictIsReportedAsConflict(t *testing.T) {
	server := recordingServer(t, http.StatusConflict, `{"code":409,"message":"an active run blocks this"}`)
	client := NewClient(ClientOptions{BaseURL: server.URL, ProductionKey: "sdk-test-token"})

	_, err := client.Workflows.CreateRun(context.Background(), "wf-1", CreateRunRequest{})
	if !IsConflict(err) {
		t.Fatalf("err = %v, want a conflict", err)
	}
}

func TestNonZeroCodeOnA2xxStatusIsStillAnError(t *testing.T) {
	server := recordingServer(t, http.StatusOK, `{"code":403,"message":"production key is required"}`)
	client := NewClient(ClientOptions{BaseURL: server.URL, ProductionKey: "sdk-test-token"})

	_, err := client.Workflows.CreateRun(context.Background(), "wf-1", CreateRunRequest{})
	var apiError *APIError
	if !errors.As(err, &apiError) {
		t.Fatalf("err = %v, want *APIError", err)
	}
	if apiError.HTTPStatus != http.StatusOK || apiError.Code != 403 {
		t.Errorf("APIError = %+v", apiError)
	}
}

func TestNonJSONErrorBodyUsesTheBodyAsTheMessage(t *testing.T) {
	server := recordingServer(t, http.StatusBadGateway, "<html>bad gateway</html>")
	client := NewClient(ClientOptions{BaseURL: server.URL, ProductionKey: "sdk-test-token"})

	_, err := client.Models.List(context.Background())
	var apiError *APIError
	if !errors.As(err, &apiError) {
		t.Fatalf("err = %v, want *APIError", err)
	}
	if apiError.HTTPStatus != http.StatusBadGateway {
		t.Errorf("HTTPStatus = %d", apiError.HTTPStatus)
	}
	if !strings.Contains(apiError.Message, "bad gateway") {
		t.Errorf("Message = %q", apiError.Message)
	}
}

func TestNonJSONSuccessBodyIsAnErrorNotASilentZeroValue(t *testing.T) {
	server := recordingServer(t, http.StatusOK, "<!doctype html>")
	client := NewClient(ClientOptions{BaseURL: server.URL, ProductionKey: "sdk-test-token"})

	_, err := client.Models.List(context.Background())
	if err == nil {
		t.Fatal("err = nil, want a decode failure")
	}
	if errors.Is(err, ErrMissingBaseURL) || errors.Is(err, ErrMissingProductionKey) {
		t.Errorf("err = %v, want a decode failure", err)
	}
}

func TestEmptyAndNullDataNeedNoPayload(t *testing.T) {
	for name, body := range map[string]string{
		"empty object": `{"code":0,"message":"ok"}`,
		"null data":    `{"code":0,"message":"ok","data":null}`,
		"empty list":   `{"code":0,"message":"ok","data":[]}`,
	} {
		t.Run(name, func(t *testing.T) {
			server := recordingServer(t, http.StatusOK, body)
			client := NewClient(ClientOptions{BaseURL: server.URL, ProductionKey: "sdk-test-token"})
			if err := client.Workflows.Delete(context.Background(), "wf-1"); err != nil {
				t.Fatalf("Delete: %v", err)
			}
			workspaces, err := client.Workspaces.List(context.Background(), ListOptions{})
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(workspaces) != 0 {
				t.Errorf("workspaces = %+v, want none", workspaces)
			}
		})
	}
}

func TestBaseURLTrailingSlashAndBasePathArePreserved(t *testing.T) {
	var seenPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":[]}`))
	}))
	t.Cleanup(server.Close)

	client := NewClient(ClientOptions{BaseURL: server.URL + "/prefix/", ProductionKey: "sdk-test-token"})
	if client.BaseURL != server.URL+"/prefix" {
		t.Errorf("BaseURL = %q, want the trailing slash trimmed", client.BaseURL)
	}
	if _, err := client.Models.List(context.Background()); err != nil {
		t.Fatalf("List: %v", err)
	}
	if seenPath != "/prefix/api/v1/seaflow/models" {
		t.Errorf("path = %q", seenPath)
	}
}

func TestBaseURLWithoutSchemeOrHostIsRejected(t *testing.T) {
	client := NewClient(ClientOptions{BaseURL: "sea-flow.seainfra.dev", ProductionKey: "sdk-test-token"})
	_, err := client.Models.List(context.Background())
	if err == nil || !strings.Contains(err.Error(), "scheme and host") {
		t.Fatalf("err = %v, want an invalid-baseURL error", err)
	}
}

// TestOneClientServesManyEndUsersConcurrently is the property an integrating
// product's server depends on: one client, many end users in flight, and no
// request ever carrying another end user's identifier.
func TestOneClientServesManyEndUsersConcurrently(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Answer with the identifier the request actually carried, so the caller
		// can compare it against the one it asked for.
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":[{"id":"` +
			r.Header.Get(EndUserHeader) + `"}]}`))
	}))
	t.Cleanup(server.Close)

	shared := NewClient(ClientOptions{BaseURL: server.URL, ProductionKey: "sdk-test-token"})

	const users, callsPerUser = 24, 5
	var waitGroup sync.WaitGroup
	failures := make(chan string, users*callsPerUser)
	for index := 0; index < users; index++ {
		endUser := fmt.Sprintf("end-user-%d", index)
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			client := shared.WithEndUser(endUser)
			for call := 0; call < callsPerUser; call++ {
				workspaces, err := client.Workspaces.List(context.Background(), ListOptions{})
				if err != nil {
					failures <- fmt.Sprintf("%s call %d: %v", endUser, call, err)
					return
				}
				if len(workspaces) != 1 || workspaces[0].ID != endUser {
					failures <- fmt.Sprintf("%s call %d reached the server as %v", endUser, call, workspaces)
					return
				}
			}
		}()
	}
	waitGroup.Wait()
	close(failures)
	for failure := range failures {
		t.Error(failure)
	}
	if shared.EndUserID != "" {
		t.Errorf("the shared client's default end user changed to %q", shared.EndUserID)
	}
}
