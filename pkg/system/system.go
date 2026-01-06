package system

import (
	"os"
	"sync"
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

func New() System {
	return &system{}
}

type system struct {
	WorkingDir string
}

func (s *system) CWD() string {
	wd, _ := Getwd()
	return wd
}
