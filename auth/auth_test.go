package auth

import (
	"context"
	"errors"
	"testing"
)

func TestNoopAuthImplementsAuthProtocols(t *testing.T) {
	noop := NewNoop()
	var authenticator Authenticator = noop
	var authorizer Authorizer = noop

	_, err := authenticator.Authenticate(context.Background(), Credentials{Token: "token"})
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured from Authenticate, got %v", err)
	}

	err = authorizer.Authorize(context.Background(), Principal{Subject: "user-1"}, Permission{
		Resource: "orders",
		Action:   "read",
	})
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured from Authorize, got %v", err)
	}
}

func TestAuthOptions(t *testing.T) {
	options := NewOptions(WithIssuer("issuer"), WithAudience("orders"))
	if options.Issuer != "issuer" {
		t.Fatalf("expected issuer, got %q", options.Issuer)
	}
	if options.Audience != "orders" {
		t.Fatalf("expected audience orders, got %q", options.Audience)
	}
}

func TestCredentialsFromHeadersParsesBearerToken(t *testing.T) {
	credentials := CredentialsFromHeaders(map[string][]string{
		"authorization": {"Bearer token-1"},
		"X-Extra":       {"value"},
	})

	if credentials.Scheme != SchemeBearer {
		t.Fatalf("expected bearer scheme, got %q", credentials.Scheme)
	}
	if credentials.Token != "token-1" {
		t.Fatalf("expected token-1, got %q", credentials.Token)
	}
	if got := HeaderValue(credentials.Header, "Authorization"); got != "Bearer token-1" {
		t.Fatalf("expected authorization header lookup, got %q", got)
	}
}

func TestPrincipalContextAndClaimsHelpers(t *testing.T) {
	claims := map[string]any{
		"sub":       "alice",
		"tenant_id": "tenant-a",
		"roles":     []any{"admin", "ops"},
		"scope":     "orders:read profile",
	}
	principal := PrincipalFromClaims(claims)

	if principal.Subject != "alice" || principal.Tenant != "tenant-a" {
		t.Fatalf("unexpected principal: %#v", principal)
	}
	if !principal.HasRole("ADMIN") {
		t.Fatalf("expected case-insensitive admin role match")
	}
	scopes := ClaimStrings(claims, ClaimScope)
	if len(scopes) != 2 || scopes[0] != "orders:read" || scopes[1] != "profile" {
		t.Fatalf("unexpected scopes: %#v", scopes)
	}

	ctx := ContextWithPrincipal(context.Background(), principal)
	got, ok := PrincipalFromContext(ctx)
	if !ok || got.Subject != "alice" {
		t.Fatalf("expected principal from context, got %#v %v", got, ok)
	}
}

func TestPermissionPolicyAuthorizer(t *testing.T) {
	policy := PermissionPolicy{
		Rules: []PermissionRule{
			{
				Permission: PermissionFor("/orders", "get"),
				Roles:      []string{"reader"},
				Decision:   Allow(),
			},
		},
		Default: Deny("no matching rule"),
	}
	authorizer := AuthorizerFromPolicy(policy)

	err := authorizer.Authorize(context.Background(), Principal{Subject: "alice", Roles: []string{"reader"}}, PermissionFor("/orders", "GET"))
	if err != nil {
		t.Fatalf("expected reader to be allowed, got %v", err)
	}
	err = authorizer.Authorize(context.Background(), Principal{Subject: "bob", Roles: []string{"writer"}}, PermissionFor("/orders", "GET"))
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("expected permission denied, got %v", err)
	}
}
