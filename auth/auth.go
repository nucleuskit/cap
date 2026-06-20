package auth

import (
	"context"
	"strings"
)

const (
	HeaderAuthorization = "Authorization"
	HeaderSignature     = "X-Nucleus-Signature"
	HeaderTimestamp     = "X-Nucleus-Timestamp"
	HeaderKeyID         = "X-Nucleus-Key-Id"
	SchemeBearer        = "Bearer"
	SchemeSignature     = "Signature"
	ActionInvoke        = "invoke"
)

const (
	ClaimSubject = "sub"
	ClaimTenant  = "tenant"
	ClaimRoles   = "roles"
	ClaimScope   = "scope"
)

type principalContextKey struct{}

type Credentials struct {
	Token  string
	Header map[string][]string
	Scheme string
	Claims map[string]any
}

type Principal struct {
	Subject string
	Tenant  string
	Roles   []string
	Claims  map[string]any
}

type Permission struct {
	Resource string
	Action   string
	Scope    string
}

type Decision struct {
	Allowed bool
	Reason  string
}

type Authenticator interface {
	Authenticate(ctx context.Context, credentials Credentials) (Principal, error)
}

type Authorizer interface {
	Authorize(ctx context.Context, principal Principal, permission Permission) error
}

func ContextWithPrincipal(ctx context.Context, principal Principal) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, principalContextKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	if ctx == nil {
		return Principal{}, false
	}
	principal, ok := ctx.Value(principalContextKey{}).(Principal)
	return principal, ok
}

func CredentialsFromHeaders(headers map[string][]string) Credentials {
	scheme, token := ParseAuthorization(HeaderValue(headers, HeaderAuthorization))
	return Credentials{
		Token:  token,
		Header: CloneHeaders(headers),
		Scheme: scheme,
	}
}

func ParseAuthorization(value string) (scheme string, token string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ""
	}
	prefix, rest, ok := strings.Cut(value, " ")
	if !ok {
		return "", value
	}
	token = strings.TrimSpace(rest)
	if strings.EqualFold(prefix, SchemeBearer) {
		return SchemeBearer, token
	}
	return strings.TrimSpace(prefix), token
}

func HeaderValue(headers map[string][]string, key string) string {
	if len(headers) == 0 || strings.TrimSpace(key) == "" {
		return ""
	}
	if values, ok := headers[key]; ok {
		if value := firstHeader(values); value != "" {
			return value
		}
	}
	for candidate, values := range headers {
		if strings.EqualFold(candidate, key) {
			return firstHeader(values)
		}
	}
	return ""
}

func CloneHeaders(headers map[string][]string) map[string][]string {
	if len(headers) == 0 {
		return nil
	}
	clone := make(map[string][]string, len(headers))
	for key, values := range headers {
		clone[key] = append([]string(nil), values...)
	}
	return clone
}

func PrincipalFromClaims(claims map[string]any) Principal {
	principal := Principal{Claims: CloneClaims(claims)}
	if subject, ok := ClaimString(claims, ClaimSubject, "subject", "user_id"); ok {
		principal.Subject = subject
	}
	if tenant, ok := ClaimString(claims, ClaimTenant, "tenant_id"); ok {
		principal.Tenant = tenant
	}
	principal.Roles = ClaimStrings(claims, ClaimRoles, "role")
	return principal
}

func CloneClaims(claims map[string]any) map[string]any {
	if len(claims) == 0 {
		return nil
	}
	clone := make(map[string]any, len(claims))
	for key, value := range claims {
		clone[key] = value
	}
	return clone
}

func ClaimString(claims map[string]any, keys ...string) (string, bool) {
	for _, key := range keys {
		value, ok := claims[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case string:
			if trimmed := strings.TrimSpace(typed); trimmed != "" {
				return trimmed, true
			}
		case []byte:
			if trimmed := strings.TrimSpace(string(typed)); trimmed != "" {
				return trimmed, true
			}
		}
	}
	return "", false
}

func ClaimStrings(claims map[string]any, keys ...string) []string {
	values := []string{}
	for _, key := range keys {
		value, ok := claims[key]
		if !ok {
			continue
		}
		values = append(values, stringsFromClaim(value)...)
	}
	return compactStrings(values)
}

func (p Principal) HasRole(role string) bool {
	for _, candidate := range p.Roles {
		if strings.EqualFold(candidate, role) {
			return true
		}
	}
	return false
}

func PermissionFor(resource string, action string) Permission {
	return Permission{
		Resource: resource,
		Action:   strings.ToUpper(strings.TrimSpace(action)),
	}
}

func InvokePermission(fullMethod string) Permission {
	return Permission{
		Resource: strings.TrimSpace(fullMethod),
		Action:   ActionInvoke,
	}
}

func firstHeader(values []string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func stringsFromClaim(value any) []string {
	switch typed := value.(type) {
	case string:
		return splitClaimStrings(typed)
	case []string:
		return typed
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			if value, ok := item.(string); ok {
				values = append(values, value)
			}
		}
		return values
	default:
		return nil
	}
}

func splitClaimStrings(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n'
	})
}

func compactStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]bool{}
	compacted := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		compacted = append(compacted, value)
	}
	return compacted
}
