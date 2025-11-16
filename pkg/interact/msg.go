package interact

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/lipgloss"
)

var (
	Output io.Writer = os.Stdout
	Input  io.Reader = os.Stdin
)

var (
	green  = lipgloss.AdaptiveColor{Light: "#02BA84", Dark: "#02BF87"}
	yellow = lipgloss.AdaptiveColor{Light: "#FFFF00", Dark: "#F1FA8C"}
	red    = lipgloss.AdaptiveColor{Light: "#FF4672", Dark: "#ED567A"}
	teal   = lipgloss.AdaptiveColor{Light: "#5A56E0", Dark: "#7571F9"}
)

func Infof(s string, args ...any) {
	_, _ = fmt.Fprintln(Output, InfoStringf(s, args...))
}

func Errorf(s string, args ...any) {
	_, _ = fmt.Fprintln(Output, ErrorStringf(s, args...))
}

func Warnf(s string, args ...any) {
	_, _ = fmt.Fprintln(Output, WarnStringf(s, args...))
}

func Successf(s string, args ...any) {
	_, _ = fmt.Fprintln(Output, SuccessStringf(s, args...))
}

func Error(s string) {
	_, _ = fmt.Fprintln(Output, ErrorString(s))
}

func Info(s string) {
	_, _ = fmt.Fprintln(Output, InfoString(s))
}

func Warn(s string) {
	_, _ = fmt.Fprintln(Output, WarnString(s))
}

func Success(s string) {
	_, _ = fmt.Fprintln(Output, InfoString(s))
}

func InfoStringf(s string, args ...any) string {
	return InfoString(fmt.Sprintf(s, args...))
}

func ErrorStringf(s string, args ...any) string {
	return ErrorString(fmt.Sprintf(s, args...))
}

func WarnStringf(s string, args ...any) string {
	return WarnString(fmt.Sprintf(s, args...))
}

func SuccessStringf(s string, args ...any) string {
	return SuccessString(fmt.Sprintf(s, args...))
}

func InfoString(s string) string {
	return lipgloss.NewStyle().Foreground(teal).Render(s)
}

func ErrorString(s string) string {
	return lipgloss.NewStyle().Foreground(red).Render(s)
}

func WarnString(s string) string {
	return lipgloss.NewStyle().Foreground(yellow).Render(s)
}

func SuccessString(s string) string {
	return lipgloss.NewStyle().Foreground(green).Render(s)
}
