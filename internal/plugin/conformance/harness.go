package conformance

import (
	"bytes"
	"path/filepath"
	"strings"

	"github.com/Section9Labs/Cartero/internal/cli"
	"github.com/Section9Labs/Cartero/internal/plugin"
	"github.com/Section9Labs/Cartero/internal/version"
)

type Result struct {
	Discovery plugin.Discovery
	Output    string
}

func PluginList(root string) (Result, error) {
	discovery, err := plugin.Discover(filepath.Join(root, "plugins"))
	if err != nil {
		return Result{}, err
	}

	var out bytes.Buffer
	cmd := cli.NewRootCmd(cli.IOStreams{
		Out: &out,
		Err: &out,
	}, version.BuildInfo())
	cmd.SetArgs([]string{"--plain", "--root", root, "plugin", "list"})
	if err := cmd.Execute(); err != nil {
		return Result{}, err
	}

	return Result{
		Discovery: discovery,
		Output:    NormalizeOutput(out.String()),
	}, nil
}

func NormalizeOutput(output string) string {
	return strings.TrimSpace(strings.ReplaceAll(output, "\r\n", "\n"))
}
