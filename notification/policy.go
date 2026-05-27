package notification

import (
	"github.com/zloevil/jet"
	"strings"
)

const (
	R = "r" // R read
	W = "w" // W write
	X = "x" // X execute
	D = "d" // D delete
)

type PermissionRequest struct {
	Resource string   // Resource requested resource
	Perm     []string // Perm requested permissions (R, W, X, D)
}

type PermissionsRequest struct {
	Items []*PermissionRequest
}

type policy struct {
	Resources   []string `json:"res,omitempty"`   // Resource code
	Permissions []string `json:"perms,omitempty"` // Permissions requested (R, W, X, D)
}

func (p *policy) Resolve() *PermissionsRequest {
	return &PermissionsRequest{
		Items: jet.Map(p.Resources, func(resource string) *PermissionRequest {
			return &PermissionRequest{
				Resource: resource,
				Perm:     p.Permissions,
			}
		}),
	}
}

type Resolver interface {
	Resolve() *PermissionsRequest
}

func Resource(permissions string, resources ...string) Resolver {
	p := &policy{
		Resources: resources,
	}
	p.Permissions = p.convertPermissions(permissions)
	return p
}

func (p *policy) convertPermissions(permissions string) []string {
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
