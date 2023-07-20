package auth

import (
	"github.com/golang-jwt/jwt/v4"
	runtimev1 "github.com/rilldata/rill/proto/gen/rill/runtime/v1"
	"github.com/rilldata/rill/runtime/server/evaluate"
)

// Claims resolves permissions for a requester.
type Claims interface {
	// Subject returns the token subject if present (usually a user or service ID)
	Subject() string
	// Can resolves system-level permissions.
	Can(p Permission) bool
	// CanInstance resolves instance-level permissions.
	CanInstance(instanceID string, p Permission) bool
	Evaluate(meta *runtimev1.ModelMeta, role string) *evaluate.EvaluatedModel
	IsRestrictedRole() bool
}

// jwtClaims implements Claims and resolve permissions based on a JWT payload.
type jwtClaims struct {
	jwt.RegisteredClaims
	System          []Permission            `json:"sys,omitempty"`
	Instances       map[string][]Permission `json:"ins,omitempty"`
	RestrictedRoles map[string]map[string]string
}

func (c *jwtClaims) Subject() string {
	return c.RegisteredClaims.Subject
}

func (c *jwtClaims) Can(p Permission) bool {
	for _, p2 := range c.System {
		if p2 == p {
			return true
		}
	}
	return false
}

func (c *jwtClaims) CanInstance(instanceID string, p Permission) bool {
	for _, p2 := range c.Instances[instanceID] {
		if p2 == p {
			return true
		}
	}
	return c.Can(p)
}

func (c *jwtClaims) Evaluate(m *runtimev1.ModelMeta, role string) *evaluate.EvaluatedModel {
	evaluatedModel := &evaluate.EvaluatedModel{}

	if !c.IsRestrictedRole() || (m.Access.Condition == "") {
		evaluatedModel.Access = true
		return evaluatedModel
	}

	attrs := c.RestrictedRoles[role]
	if attrs == nil {
		return evaluatedModel
	}

	expr, err := evaluate.Template(m.Access.Condition, attrs)
	// TODO: log error
	if err != nil {
		return nil
	}

	// evaluate model access
	result, err := evaluate.Expr(expr)
	if err != nil {
		return nil
	}
	evaluatedModel.Access = result

	// evaluate filter
	if m.Access.Filter != "" {
		evaluatedModel.Filter, err = evaluate.Template(m.Access.Filter, attrs)
		if err != nil {
			return nil
		}
	}
	// TODO evaluate include/exclude columns

	return evaluatedModel
}

func (c *jwtClaims) IsRestrictedRole() bool {
	return len(c.RestrictedRoles) > 0
}

// openClaims implements Claims and allows all actions.
// It is used for servers with auth disabled.
type openClaims struct{}

func (c openClaims) Subject() string {
	return ""
}

func (c openClaims) Can(p Permission) bool {
	return true
}

func (c openClaims) CanInstance(instanceID string, p Permission) bool {
	return true
}

// anonClaims imeplements Claims with no permissions.
// It is used for unauthorized requests when auth is enabled.
type anonClaims struct{}

func (c anonClaims) Subject() string {
	return ""
}

func (c anonClaims) Can(p Permission) bool {
	return false
}

func (c anonClaims) CanInstance(instanceID string, p Permission) bool {
	return false
}

func (c anonClaims) Evaluate(m *runtimev1.ModelMeta, role string) *evaluate.EvaluatedModel {
	return nil
}

func (c anonClaims) IsRestrictedRole() bool {
	return false
}
