package artifact

import "fmt"

func goURL(ver string, os, arch string) string {
	var ext string
	if os == "windows" {
		ext = ".zip"
	} else {
		ext = ".tar.gz"
	}
	return fmt.Sprintf("https://go.dev/dl/go%s.%s-%s%s", ver, os, arch, ext)
}

var _ Artifact = (*Go)(nil)

type Go struct {
	Version string
}

func (g *Go) URL(os, arch string) (string, error) {
	return goURL(g.Version, os, arch), nil
}

func (g *Go) Name() string {
	return "go"
}
