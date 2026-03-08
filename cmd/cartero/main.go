package main

import (
	"fmt"
	"os"

	"github.com/Section9Labs/Cartero/internal/cli"
	"github.com/Section9Labs/Cartero/internal/version"
)

func main() {
	if err := cli.Execute(cli.IOStreams{
		In:  os.Stdin,
		Out: os.Stdout,
		Err: os.Stderr,
	}, version.BuildInfo()); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
