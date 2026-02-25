package main

import (
	"os"

	"github.com/fanwenlin/codex-go-sdk/cmd/codex-orchestrator/cli"
)

func main() {
	exitCode := cli.RunCli(os.Args[1:], cli.CliIo{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
	os.Exit(exitCode)
}
