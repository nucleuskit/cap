package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrUnauthenticated  = errors.New("unauthenticated")
	ErrPermissionDenied = errors.New("permission denied")
)

type DeniedError struct {
	Reason string
}

func (e DeniedError) Error() string {
	if strings.TrimSpace(e.Reason) == "" {
		return ErrPermissionDenied.Error()
	}
	return fmt.Sprintf("%s: %s", ErrPermissionDenied, e.Reason)
}

func (e DeniedError) Unwrap() error {
	return ErrPermissionDenied
}

type Policy interface {
	Evaluate(ctx context.Context, principal Principal, permission Permission) Decision
}

type PolicyFunc func(ctx context.Context, principal Principal, permission Permission) Decision

func (fn PolicyFunc) Evaluate(ctx context.Context, principal Principal, permission Permission) Decision {
	if fn == nil {
		return Decision{Allowed: false, Reason: "policy not configured"}
	}
	return fn(ctx, principal, permission)
}

type AuthorizerFunc func(ctx context.Context, principal Principal, permission Permission) error

func (fn AuthorizerFunc) Authorize(ctx context.Context, principal Principal, permission Permission) error {
	if fn == nil {
		return ErrNotConfigured
	}
	return fn(ctx, principal, permission)
}

type PermissionRule struct {
	Permission Permission
	Roles      []string
	Subjects   []string
	Decision   Decision
}

type PermissionPolicy struct {
	Rules   []PermissionRule
	Default Decision
}

func (p PermissionPolicy) Evaluate(_ context.Context, principal Principal, permission Permission) Decision {
	for _, rule := range p.Rules {
		if rule.matches(principal, permission) {
			return normalizeDecision(rule.Decision)
		}
	}
	return normalizeDecision(p.Default)
}

func AuthorizerFromPolicy(policy Policy) Authorizer {
	return AuthorizerFunc(func(ctx context.Context, principal Principal, permission Permission) error {
		if policy == nil {
			return ErrNotConfigured
		}
		decision := normalizeDecision(policy.Evaluate(ctx, principal, permission))
		if decision.Allowed {
			return nil
		}
		return DeniedError{Reason: decision.Reason}
	})
}

func Allow() Decision {
	return Decision{Allowed: true}
}

func Deny(reason string) Decision {
	return Decision{Allowed: false, Reason: reason}
}

func (r PermissionRule) matches(principal Principal, permission Permission) bool {
	if !matchPermission(r.Permission, permission) {
		return false
	}
	if len(r.Subjects) > 0 && !matchAny(r.Subjects, principal.Subject) {
		return false
	}
	if len(r.Roles) == 0 {
		return true
	}
	for _, role := range r.Roles {
		if principal.HasRole(role) || role == "*" {
			return true
		}
	}
	return false
}

func matchPermission(rule Permission, permission Permission) bool {
	return matchField(rule.Resource, permission.Resource) &&
		matchField(rule.Action, permission.Action) &&
		matchField(rule.Scope, permission.Scope)
}

func matchField(rule string, value string) bool {
	rule = strings.TrimSpace(rule)
	if rule == "" || rule == "*" {
		return true
	}
	return strings.EqualFold(rule, strings.TrimSpace(value))
}

func matchAny(candidates []string, value string) bool {
	for _, candidate := range candidates {
		if matchField(candidate, value) {
			return true
		}
	}
	return false
}

func normalizeDecision(decision Decision) Decision {
	if decision.Allowed {
		return decision
	}
	if strings.TrimSpace(decision.Reason) == "" {
		decision.Reason = ErrPermissionDenied.Error()
	}
	return decision
}
