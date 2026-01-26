package codex

import (
	"github.com/fanwenlin/codex-go-sdk/types"
)

// Exec defines the interface for executing codex commands
type Exec interface {
	Run(args CodexExecArgs) <-chan ExecResult
}

// Codex is the main class for interacting with the Codex agent.
// Use the StartThread() method to start a new thread or ResumeThread() to resume a previously started thread.
type Codex struct {
	exec    Exec
	options types.CodexOptions
}

// NewCodex creates a new Codex client.
func NewCodex(options types.CodexOptions) *Codex {
	var exec Exec
	if options.CodexPathOverride != "" {
		exec = NewCodexExec(options.CodexPathOverride, options.Env)
	} else {
		// Try to find codex in PATH or parent project
		exec = NewCodexExec("", options.Env)
	}
	if options.Verbose {
		if codexExec, ok := exec.(*CodexExec); ok {
			codexExec.EnableVerbose(options.VerboseWriter)
		}
	}
	return &Codex{
		exec:    exec,
		options: options,
	}
}

// NewCodexWithExec creates a new Codex client with a custom Exec implementation.
// This is intended for testing purposes.
func NewCodexWithExec(exec Exec, options types.CodexOptions) *Codex {
	if options.Verbose {
		if codexExec, ok := exec.(*CodexExec); ok {
			codexExec.EnableVerbose(options.VerboseWriter)
		}
	}
	return &Codex{
		exec:    exec,
		options: options,
	}
}

// StartThread starts a new conversation with an agent.
// Returns a new thread instance.
func (c *Codex) StartThread(options types.ThreadOptions) *Thread {
	return newThread(c.exec, c.options, options, nil)
}

// ResumeThread resumes a conversation with an agent based on the thread ID.
// Threads are persisted in ~/.codex/sessions.
//
// Parameters:
//   - id: The ID of the thread to resume
//   - options: Options for the thread
//
// Returns a new thread instance.
func (c *Codex) ResumeThread(id string, options types.ThreadOptions) *Thread {
	return newThread(c.exec, c.options, options, &id)
}
