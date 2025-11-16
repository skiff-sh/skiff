package interact

import (
	"context"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/x/term"
)

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

func Prompt(ctx context.Context, prompt string) (val string, err error) {
	err = NewHuhForm(NewHuhGroup(huh.NewInput().Title(prompt).Value(&val))).RunWithContext(ctx)
	if err != nil {
		return "", err
	}

	return val, nil
}

func Confirm(ctx context.Context, prompt string) (val bool, err error) {
	err = NewHuhForm(NewHuhGroup(huh.NewConfirm().Title(prompt).Value(&val))).RunWithContext(ctx)
	if err != nil {
		return false, err
	}

	return val, nil
}
