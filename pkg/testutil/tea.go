package testutil

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/skiff-sh/skiff/pkg/collection"
)

const (
	KeyUp         = "\x1b[A"
	KeyDown       = "\x1b[B"
	KeyRight      = "\x1b[C"
	KeyLeft       = "\x1b[D"
	KeyShiftUp    = "\x1b[1;2A"
	KeyShiftDown  = "\x1b[1;2B"
	KeyShiftRight = "\x1b[1;2C"
	KeyShiftLeft  = "\x1b[1;2D"
)

type TeaWaitCond = func(b []byte) bool

func WaitFormDone(f *huh.Form) TeaWaitCond {
	return func(b []byte) bool {
		return f.State != huh.StateNormal
	}
}

func WaitRenderContains(s string) TeaWaitCond {
	return func(b []byte) bool {
		return strings.Contains(string(b), s)
	}
}

type TeaInput interface {
	Msg() tea.Msg
}

type TeaInputType interface {
	string | tea.KeyType
}

type TeaInputs []TeaInput

func (t TeaInputs) ToMsg() []tea.Msg {
	return collection.Map(t, func(e TeaInput) tea.Msg {
		return e.Msg()
	})
}

// SendTo sends all inputs to the program. If running with e2e, it's recommended to set timeBetween as the
// update ticks don't have enough time to process if sent all at once.
func (t TeaInputs) SendTo(prog teatest.Program, timeBetween time.Duration) {
	if timeBetween > 0 {
		time.Sleep(timeBetween)
	}
	for _, v := range t.ToMsg() {
		prog.Send(v)
		if timeBetween > 0 {
			time.Sleep(timeBetween)
		}
	}
}

// Inputs helper method to create TeaInputs. Only TeaInputType's are accepted everything else is ignored.
func Inputs(t ...any) TeaInputs {
	out := make(TeaInputs, 0, len(t))
	for _, v := range t {
		switch typ := v.(type) {
		case string:
			out = append(out, &teaInput{
				Str: typ,
			})
		case tea.KeyType:
			out = append(out, &teaInput{
				Key: typ,
			})
		}
	}

	return out
}

var _ TeaInput = (*teaInput)(nil)

type teaInput struct {
	Str string
	Key tea.KeyType
}

func (t *teaInput) Msg() tea.Msg {
	if t.Str == "" {
		return tea.KeyMsg{
			Type: t.Key,
		}
	}

	return tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune(t.Str),
	}
}

func NewFormTest(f *huh.Form) tea.Model {
	return &model{
		Form: f,
	}
}

type TickMsg time.Time

type model struct{ Form *huh.Form }

func (m *model) Init() tea.Cmd {
	return tea.Batch(tea.Tick(5*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	}), m.Form.Init())
}
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.Form.State != huh.StateNormal {
		return m, tea.Quit
	}

	_, cmd := m.Form.Update(msg)

	return m, cmd
}
func (m *model) View() string { return m.Form.View() }
