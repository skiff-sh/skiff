package interact

import (
	"context"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/x/term"
)

var DefaultFormRunner = func(ctx context.Context, f *huh.Form) error {
	return f.RunWithContext(ctx)
}

func NewHuhForm(groups ...*huh.Group) *huh.Form {
	return huh.NewForm(groups...).WithOutput(Output).WithInput(Input).WithAccessible(!IsTerminal())
}

func NewHuhGroup(fields ...huh.Field) *huh.Group {
	return huh.NewGroup(fields...).
		WithShowHelp(true).
		WithShowErrors(true)
}

func IsTerminal() bool {
	return term.IsTerminal(os.Stdin.Fd())
}

func Confirm(ctx context.Context, fact func(c *huh.Confirm) *huh.Confirm) bool {
	var val bool
	err := DefaultFormRunner(ctx, NewHuhForm(NewHuhGroup(fact(huh.NewConfirm()).Value(&val))))
	if err != nil {
		return false
	}
	return val
}
