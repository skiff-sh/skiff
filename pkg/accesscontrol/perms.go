package accesscontrol

import (
	"fmt"
	"sync"

	"github.com/skiff-sh/api/go/skiff/registry/v1alpha1"
)

// PermDescriptions user-friendly descriptions as to what each permission is allowing. These descriptions should start with lowercase and should not end with punctuation.
var PermDescriptions = sync.OnceValue[map[v1alpha1.PackagePermissions_Plugin]string](
	func() map[v1alpha1.PackagePermissions_Plugin]string {
		out := map[v1alpha1.PackagePermissions_Plugin]string{}

		for _, v := range v1alpha1.PackagePermissions_Plugin_value {
			perm := v1alpha1.PackagePermissions_Plugin(v)
			// doing it this way to force explicit descriptions to be written on every API upgrade.
			switch perm {
			case v1alpha1.PackagePermissions_all:
				out[perm] = "access to all resources"
			case v1alpha1.PackagePermissions_cwd_ro:
				out[perm] = "read access to the CWD and all subdirectories"
			}
		}

		return out
	},
)

func PermUsageListPretty(perms []v1alpha1.PackagePermissions_Plugin) []string {
	out := make([]string, 0, len(perms))
	for _, v := range perms {
		out = append(out, fmt.Sprintf("* %s: Grant plugins %s.", v, PermDescriptions()[v]))
	}
	return out
}

func AllPerms() []v1alpha1.PackagePermissions_Plugin {
	out := make([]v1alpha1.PackagePermissions_Plugin, 0, len(v1alpha1.PackagePermissions_Plugin_value))
	for k := range v1alpha1.PackagePermissions_Plugin_name {
		out = append(out, v1alpha1.PackagePermissions_Plugin(k))
	}
	return out
}
