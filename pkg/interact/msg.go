package interact

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/lipgloss"
)

var Writer io.Writer = os.Stdout

var (
	green  = lipgloss.AdaptiveColor{Light: "#02BA84", Dark: "#02BF87"}
	yellow = lipgloss.AdaptiveColor{Light: "#FFFF00", Dark: "#F1FA8C"}
	red    = lipgloss.AdaptiveColor{Light: "#FF4672", Dark: "#ED567A"}
	teal   = lipgloss.AdaptiveColor{Light: "#5A56E0", Dark: "#7571F9"}
)

func Infof(s string, args ...any) {
	_, _ = fmt.Fprintln(Writer, InfoStringf(s, args...))
}

func Errorf(s string, args ...any) {
	_, _ = fmt.Fprintln(Writer, ErrorStringf(s, args...))
}

func Warnf(s string, args ...any) {
	_, _ = fmt.Fprintln(Writer, WarnStringf(s, args...))
}

func Successf(s string, args ...any) {
	_, _ = fmt.Fprintln(Writer, SuccessStringf(s, args...))
}

func Error(s string) {
	_, _ = fmt.Fprintln(Writer, ErrorString(s))
}

func Info(s string) {
	_, _ = fmt.Fprintln(Writer, InfoString(s))
}

func Warn(s string) {
	_, _ = fmt.Fprintln(Writer, WarnString(s))
}

func Success(s string) {
	_, _ = fmt.Fprintln(Writer, InfoString(s))
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
