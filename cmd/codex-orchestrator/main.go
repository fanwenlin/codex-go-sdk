package main

import (
	"os"

	//nolint:depguard // Internal SDK imports are allowed
	"github.com/fanwenlin/codex-go-sdk/cmd/codex-orchestrator/cli"
)

func main() {
	exitCode := cli.RunCli(os.Args[1:], cli.CliIo{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
	os.Exit(exitCode)
}
