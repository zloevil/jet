package http

import (
	"context"
	"net/http"
	"strings"
)

const (
	R = "r" // R read
	W = "w" // W write
	X = "x" // X execute
	D = "d" // D delete
)

type AuthorizationResource struct {
	Resource    string   `json:"res,omitempty"`   // Resource code
	Permissions []string `json:"perms,omitempty"` // Permissions requested (R, W, X, D)
}

// ResourcePolicyManager accumulates mapping between URLs and requested resources and then convert it to Authorization request
type ResourcePolicyManager interface {
	// RegisterResourceMapping maps routeId and resource policies
	RegisterResourceMapping(routeId string, policies ...ResourcePolicy)
	// GetRequestedResources resolves policies and retrieves accumulated resources requested to be authorized
	GetRequestedResources(ctx context.Context, routeId string, r *http.Request) ([]*AuthorizationResource, error)
}

type resourcePolicyManager struct {
	routePoliciesMap map[string][]ResourcePolicy
}

type ResourcePolicy interface {
	// Resolve determines needed authorization resources for the given request
	Resolve(ctx context.Context, r *http.Request) (*AuthorizationResource, error)
}

type ConditionFn func(context.Context, *http.Request) (bool, error)

type ResourcePolicyBuilder struct {
	resource    string
	permissions []string
	conditions  []ConditionFn
}

func NewResourcePolicyManager() ResourcePolicyManager {
	return &resourcePolicyManager{
		routePoliciesMap: map[string][]ResourcePolicy{},
	}
}

func (s *resourcePolicyManager) RegisterResourceMapping(routeId string, policies ...ResourcePolicy) {
	s.routePoliciesMap[routeId] = policies
}

func (s *resourcePolicyManager) GetRequestedResources(ctx context.Context, routeId string, r *http.Request) ([]*AuthorizationResource, error) {

	var resources []*AuthorizationResource
	var codes []string

	// go through policies registered for route and resolve resources
	if policies, ok := s.routePoliciesMap[routeId]; ok {
		for _, policy := range policies {
			resource, err := policy.Resolve(ctx, r)
			if err != nil {
				return nil, err
			}
			if resource == nil {
				continue
			}
			resources = append(resources, resource)
			codes = append(codes, resource.Resource)
		}
	}

	return resources, nil
}

func Resource(resource string, permissions string) *ResourcePolicyBuilder {
	b := &ResourcePolicyBuilder{
		resource:   resource,
		conditions: []ConditionFn{},
	}
	b.permissions = b.convertPermissions(permissions)
	return b
}

// convertPermissions converts permissions from "rwxd" string to []string{"r", w", "x", "d"}
func (a *ResourcePolicyBuilder) convertPermissions(permissions string) []string {
	var res []string
	s := strings.ToLower(permissions)
	if strings.Contains(s, R) {
		res = append(res, R)
	}
	if strings.Contains(s, W) {
		res = append(res, W)
	}
	if strings.Contains(s, X) {
		res = append(res, X)
	}
	if strings.Contains(s, D) {
		res = append(res, D)
	}
	return res
}

func (a *ResourcePolicyBuilder) When(f ...ConditionFn) *ResourcePolicyBuilder {
	a.conditions = append(a.conditions, f...)
	return a
}

func (a *ResourcePolicyBuilder) WhenNot(f ...ConditionFn) *ResourcePolicyBuilder {
	for _, ff := range f {
		fn := ff
		a.conditions = append(a.conditions, func(c context.Context, r *http.Request) (bool, error) { res, err := fn(c, r); return !res, err })
	}
	return a
}

func (a *ResourcePolicyBuilder) Resolve(ctx context.Context, r *http.Request) (*AuthorizationResource, error) {
	// check conditions
	for _, cond := range a.conditions {
		if condRes, err := cond(ctx, r); err != nil {
			return nil, err
		} else {
			if !condRes {
				return nil, nil
			}
		}
	}

	return &AuthorizationResource{
		Resource:    a.resource,
		Permissions: a.permissions,
	}, nil

}

func (a *ResourcePolicyBuilder) B() ResourcePolicy {
	return a
}
