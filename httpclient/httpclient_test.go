package httpclient

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	captransport "github.com/nucleuskit/cap/transport"
)

func TestNoopClientImplementsClient(t *testing.T) {
	var _ Client = NewNoop()

	_, err := NewNoop().Do(context.Background(), Request{
		Method: http.MethodGet,
		URL:    "https://example.com",
	})
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}
}

func TestHTTPClientOptions(t *testing.T) {
	dialer := optionDialer{}
	policy := func(ctx context.Context, target captransport.Target) (captransport.Target, error) {
		target.Metadata = captransport.MergeMetadata(target.Metadata, map[string]string{"policy": "applied"})
		return target, nil
	}
	hook := HookFunc(func(context.Context, Event) error { return nil })
	options := NewOptions(
		WithTimeout(3*time.Second),
		WithUserAgent("nucleus-test"),
		WithBaseURL("https://api.example.com"),
		WithHeader("X-App", "nucleus"),
		WithTransportDialer(dialer),
		WithTransportTargetPolicy(policy),
		WithHooks(hook),
	)
	if options.Timeout != 3*time.Second {
		t.Fatalf("expected timeout 3s, got %s", options.Timeout)
	}
	if options.UserAgent != "nucleus-test" {
		t.Fatalf("expected user agent nucleus-test, got %q", options.UserAgent)
	}
	if options.BaseURL != "https://api.example.com" {
		t.Fatalf("expected base url, got %q", options.BaseURL)
	}
	if options.Headers["X-App"] != "nucleus" {
		t.Fatalf("expected default header")
	}
	if options.TransportDialer != dialer {
		t.Fatalf("expected transport dialer option")
	}
	target, err := options.TransportTargetPolicy(context.Background(), captransport.Target{
		Metadata: captransport.Metadata{"route": "orders"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if target.Metadata["route"] != "orders" || target.Metadata["policy"] != "applied" {
		t.Fatalf("unexpected transport target metadata: %#v", target.Metadata)
	}
	if len(options.Hooks) != 1 {
		t.Fatalf("expected one hook, got %d", len(options.Hooks))
	}
	options.Hooks[0] = nil
	next := NewOptions(WithHooks(hook))
	if len(next.Hooks) != 1 || next.Hooks[0] == nil {
		t.Fatalf("expected options to clone hooks")
	}
}

func TestRequestHelpers(t *testing.T) {
	request, err := JSON(http.MethodPost, "/orders", map[string]any{"id": "1"})
	if err != nil {
		t.Fatal(err)
	}
	if request.ContentType != ContentTypeJSON {
		t.Fatalf("expected json content type, got %q", request.ContentType)
	}
	form := Form(http.MethodPost, "/orders", url.Values{"id": []string{"1"}})
	if string(form.Body) != "id=1" {
		t.Fatalf("expected encoded form body, got %q", string(form.Body))
	}
	joined, err := JoinURL("https://api.example.com/v1/", "orders")
	if err != nil {
		t.Fatal(err)
	}
	if joined != "https://api.example.com/v1/orders" {
		t.Fatalf("unexpected joined url: %s", joined)
	}
}

func TestStructTagValuesBuildQueryAndFormRequests(t *testing.T) {
	type listRequest struct {
		Status []string `query:"status"`
		Limit  int      `query:"limit"`
		Active bool     `query:"active"`
		Ignore string   `query:"-"`
	}
	queryRequest, err := Query(http.MethodGet, "/orders?cursor=old", listRequest{
		Status: []string{"paid", "pending"},
		Limit:  50,
		Active: true,
		Ignore: "skip",
	})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(queryRequest.URL)
	if err != nil {
		t.Fatal(err)
	}
	values := parsed.Query()
	if values.Get("cursor") != "old" || values.Get("limit") != "50" || values.Get("active") != "true" {
		t.Fatalf("unexpected query values: %s", parsed.RawQuery)
	}
	if got := values["status"]; len(got) != 2 || got[0] != "paid" || got[1] != "pending" {
		t.Fatalf("expected repeated status values, got %#v", got)
	}
	if values.Get("Ignore") != "" {
		t.Fatalf("expected ignored field to be omitted: %s", parsed.RawQuery)
	}

	type createRequest struct {
		Name  string `form:"name"`
		Count int64  `form:"count"`
		Dry   bool   `form:"dry_run"`
	}
	formRequest, err := FormStruct(http.MethodPost, "/orders", createRequest{
		Name:  "coffee",
		Count: 3,
		Dry:   false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if formRequest.ContentType != ContentTypeForm {
		t.Fatalf("expected form content type, got %q", formRequest.ContentType)
	}
	body, err := url.ParseQuery(string(formRequest.Body))
	if err != nil {
		t.Fatal(err)
	}
	if body.Get("name") != "coffee" || body.Get("count") != "3" || body.Get("dry_run") != "false" {
		t.Fatalf("unexpected form body: %s", string(formRequest.Body))
	}
}

func TestStructTagValuesRejectUnsupportedFields(t *testing.T) {
	type request struct {
		Nested struct{} `query:"nested"`
	}
	_, err := QueryValues(request{})
	if err == nil || !strings.Contains(err.Error(), "unsupported query field") {
		t.Fatalf("expected unsupported field error, got %v", err)
	}
}

func TestResponseDecodeJSON(t *testing.T) {
	var payload struct {
		ID string `json:"id"`
	}
	if err := (Response{Body: []byte(`{"id":"42"}`)}).DecodeJSON(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.ID != "42" {
		t.Fatalf("expected decoded id 42, got %q", payload.ID)
	}

	for name, tc := range map[string]struct {
		response Response
		target   any
		want     string
	}{
		"nil target":   {response: Response{Body: []byte(`{"id":"42"}`)}, target: nil, want: "nil target"},
		"empty body":   {response: Response{}, target: &payload, want: "empty body"},
		"invalid json": {response: Response{Body: []byte(`{"id":`)}, target: &payload, want: "invalid JSON"},
	} {
		t.Run(name, func(t *testing.T) {
			err := tc.response.DecodeJSON(tc.target)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q error, got %v", tc.want, err)
			}
		})
	}
}

func TestEventCloneProtectsMetadata(t *testing.T) {
	event := Event{Metadata: map[string]string{"route": "orders"}}
	clone := event.Clone()
	clone.Metadata["route"] = "mutated"
	if event.Metadata["route"] != "orders" {
		t.Fatalf("expected original metadata to be isolated, got %#v", event.Metadata)
	}
}

func TestRetryPolicyDefaults(t *testing.T) {
	policy := RetryPolicy{MaxAttempts: 3}
	if policy.Attempts() != 3 {
		t.Fatalf("expected 3 attempts")
	}
	if !policy.ShouldRetry(http.StatusBadGateway, nil, 1) {
		t.Fatalf("expected bad gateway retry")
	}
	if policy.ShouldRetry(http.StatusBadGateway, nil, 3) {
		t.Fatalf("expected retry to stop at max attempts")
	}
	if policy.ShouldRetry(http.StatusNotFound, errors.New("not found"), 1) {
		t.Fatalf("expected 4xx response errors not to retry")
	}
}

func TestBreakerFailureClassifiesOutboundErrors(t *testing.T) {
	networkErr := errors.New("connection refused")
	for name, tc := range map[string]struct {
		statusCode int
		err        error
		want       bool
	}{
		"4xx status":     {statusCode: http.StatusNotFound, want: false},
		"408 status":     {statusCode: http.StatusRequestTimeout, want: false},
		"429 status":     {statusCode: http.StatusTooManyRequests, want: false},
		"5xx status":     {statusCode: http.StatusBadGateway, want: true},
		"network error":  {err: networkErr, want: true},
		"timeout error":  {err: context.DeadlineExceeded, want: true},
		"successful 2xx": {statusCode: http.StatusOK, want: false},
	} {
		t.Run(name, func(t *testing.T) {
			got := BreakerFailure(tc.statusCode, tc.err) != nil
			if got != tc.want {
				t.Fatalf("expected breaker failure=%v, got %v", tc.want, got)
			}
		})
	}
}

type optionDialer struct{}

func (optionDialer) DialContext(context.Context, captransport.Target) (net.Conn, error) {
	return nil, nil
}
