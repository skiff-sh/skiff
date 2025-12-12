package accesscontrol

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/skiff-sh/api/go/skiff/registry/v1alpha1"

	"github.com/skiff-sh/skiff/pkg/interact"
)

type PluginAccessPolicy struct {
	granted []v1alpha1.PackagePermissions_Plugin
	indexed map[v1alpha1.PackagePermissions_Plugin]bool
}

func NewPluginAccessPolicy(granted []v1alpha1.PackagePermissions_Plugin) *PluginAccessPolicy {
	indexed := map[v1alpha1.PackagePermissions_Plugin]bool{}
	for _, v := range granted {
		indexed[v] = true
	}
	return &PluginAccessPolicy{
		granted: granted,
		indexed: indexed,
	}
}

func (p *PluginAccessPolicy) Diff(perms ...v1alpha1.PackagePermissions_Plugin) []v1alpha1.PackagePermissions_Plugin {
	if p.indexed[v1alpha1.PackagePermissions_all] {
		return nil
	}

	var denied []v1alpha1.PackagePermissions_Plugin
	for i := range perms {
		v := perms[i]
		_, ok := p.indexed[v]
		if !ok {
			denied = append(denied, v)
		}
	}
	return denied
}

func (p *PluginAccessPolicy) Grant(perms ...v1alpha1.PackagePermissions_Plugin) {
	for _, v := range perms {
		_, ok := p.indexed[v]
		if ok {
			continue
		}

		p.granted = append(p.granted, v)
		p.indexed[v] = true
	}
}

func (p *PluginAccessPolicy) Authorize(perm v1alpha1.PackagePermissions_Plugin) bool {
	return p.indexed[perm] || p.indexed[v1alpha1.PackagePermissions_all]
}

type Granter interface {
	RequestAccess(ctx context.Context, packageName string, requests []v1alpha1.PackagePermissions_Plugin) bool
}

var _ Granter = (*TerminalGranter)(nil)

type TerminalGranter struct {
}

func NewTerminalGranter() *TerminalGranter {
	return &TerminalGranter{}
}

func (t *TerminalGranter) RequestAccess(
	ctx context.Context,
	packageName string,
	requests []v1alpha1.PackagePermissions_Plugin,
) bool {
	permsStr := strings.Join(PermUsageListPretty(requests), "\n")
	grant := interact.Confirm(ctx, func(c *huh.Confirm) *huh.Confirm {
		return c.Title(fmt.Sprintf("Package %s needs access to:\n\n%s\n", packageName, permsStr)).
			Affirmative("Grant").
			Negative("Deny")
	})
	return grant
}
