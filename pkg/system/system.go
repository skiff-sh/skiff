package system

import (
	"os"
	"sync"

	"github.com/skiff-sh/api/go/skiff/registry/v1alpha1"

	"github.com/skiff-sh/skiff/pkg/accesscontrol"
)

var (
	// A stored version of os.Getwd(). Set on the first Get* call.
	cwd string
)

func Getwd() (string, error) {
	if err := initer(); err != nil {
		return "", err
	}

	return cwd, nil
}

func Setwd(wd string) {
	cwd = wd
}

var initer = sync.OnceValue[error](func() error {
	var err error
	cwd, err = os.Getwd()
	if err != nil {
		return err
	}

	return nil
})

// System abstraction for the user's system.
type System interface {
	// CWD returns the user's current working directory. If empty, permission was not granted.
	CWD() string
}

// Mediator provides the capabilities of the System but mediated by a policy.
type Mediator interface {
	MediatedSystem(policy *accesscontrol.PluginAccessPolicy) System
}

// NewMediator constructor for Mediator.
func NewMediator() Mediator {
	out := &mediator{}

	return out
}

type mediator struct {
}

func (m *mediator) MediatedSystem(policy *accesscontrol.PluginAccessPolicy) System {
	return &system{
		Policy: policy,
	}
}

type system struct {
	Policy     *accesscontrol.PluginAccessPolicy
	WorkingDir string
}

func (s *system) CWD() string {
	if s.Policy.Authorize(v1alpha1.PackagePermissions_cwd_ro) {
		wd, _ := Getwd()
		return wd
	}
	return ""
}
