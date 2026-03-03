package cli

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/fanwenlin/codex-go-sdk/cmd/codex-orchestrator/orchestrator"
)

// CliOptions contains CLI options.
//
//nolint:revive // Name stutter is acceptable for exported API
type CliOptions struct {
	DocDir              string
	SkillsDir           string
	MaxFileBytes        int
	MaxTotalBytes       int
	Help                bool
	Verbose             bool
	DisableGlobalSkills bool
	Quiet               bool // Disable progress output
}

// CliIo contains I/O streams for the CLI.
//
//nolint:revive // Name stutter is acceptable for exported API
type CliIo struct {
	Stdout io.Writer
	Stderr io.Writer
}

// RunCli runs the CLI with the given arguments.
func RunCli(args []string, cliIo CliIo) int {
	options, errors := parseArgs(args)

	if options.Help {
		_, _ = cliIo.Stdout.Write([]byte(renderUsage()))
		return 0
	}

	if len(errors) > 0 {
		_, _ = cliIo.Stderr.Write([]byte(joinErrors(errors) + "\n"))
		_, _ = cliIo.Stderr.Write([]byte(renderUsage()))
		return 1
	}

	if options.DocDir == "" {
		_, _ = cliIo.Stderr.Write([]byte("Missing required document directory.\n"))
		_, _ = cliIo.Stderr.Write([]byte(renderUsage()))
		return 1
	}

	progressWriter := cliIo.Stdout
	if options.Quiet {
		progressWriter = nil // nil means discard in orchestrator
	}

	orchOptions := orchestrator.OrchestratorOptions{
		DocDir:              options.DocDir,
		IncludeSkills:       true,
		DisableGlobalSkills: options.DisableGlobalSkills,
		SkillsDir:           options.SkillsDir,
		MaxFileBytes:        options.MaxFileBytes,
		MaxTotalBytes:       options.MaxTotalBytes,
		Verbose:             options.Verbose,
		VerboseWriter:       cliIo.Stderr,
		ProgressWriter:      progressWriter,
	}

	result, err := orchestrator.RunOrchestrator(orchOptions)
	if err != nil {
		_, _ = cliIo.Stderr.Write([]byte(fmtError(err)))
		return 1
	}

	_, _ = cliIo.Stdout.Write([]byte(result.FinalResponse + "\n"))
	return 0
}

//nolint:gocognit,nestif // Argument parsing requires sequential steps with nested conditions
func parseArgs(args []string) (CliOptions, []error) {
	var options CliOptions
	var errors []error

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if arg == "--docs" || arg == "-d" {
			if i+1 >= len(args) {
				errors = append(errors, fmt.Errorf("missing value for %s", arg))
			} else {
				options.DocDir = args[i+1]
				i++
			}
			continue
		}

		if arg == "--skills" || arg == "-s" {
			if i+1 >= len(args) {
				errors = append(errors, fmt.Errorf("missing value for %s", arg))
			} else {
				options.SkillsDir = args[i+1]
				i++
			}
			continue
		}

		if arg == "--max-file-bytes" {
			if i+1 >= len(args) {
				//nolint:perfsprint // Variable shadowing prevents errors.New usage
				errors = append(errors, fmt.Errorf("missing value for --max-file-bytes"))
			} else {
				value, err := strconv.Atoi(args[i+1])
				if err != nil || value <= 0 {
					errors = append(errors, fmt.Errorf("invalid --max-file-bytes value: %s", args[i+1]))
				} else {
					options.MaxFileBytes = value
					i++
				}
			}
			continue
		}

		if arg == "--max-total-bytes" {
			if i+1 >= len(args) {
				//nolint:perfsprint // Variable shadowing prevents errors.New usage
				errors = append(errors, fmt.Errorf("missing value for --max-total-bytes"))
			} else {
				value, err := strconv.Atoi(args[i+1])
				if err != nil || value <= 0 {
					errors = append(errors, fmt.Errorf("invalid --max-total-bytes value: %s", args[i+1]))
				} else {
					options.MaxTotalBytes = value
					i++
				}
			}
			continue
		}

		if arg == "--help" || arg == "-h" {
			options.Help = true
			continue
		}

		if arg == "--verbose" || arg == "-v" {
			options.Verbose = true
			continue
		}

		if arg == "--disable-global-skills" {
			options.DisableGlobalSkills = true
			continue
		}

		if arg == "--quiet" || arg == "-q" {
			options.Quiet = true
			continue
		}

		if arg != "" && !startsWith(arg, "-") && options.DocDir == "" {
			options.DocDir = arg
			continue
		}

		if arg != "" && !startsWith(arg, "-") {
			errors = append(errors, fmt.Errorf("unknown argument: %s", arg))
			continue
		}

		if startsWith(arg, "-") {
			errors = append(errors, fmt.Errorf("unknown argument: %s", arg))
		}
	}

	return options, errors
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func renderUsage() string {
	return `Usage:
  codex-orchestrator <doc-dir> [options]

Arguments:
  <doc-dir>                Document directory containing task requirements

Options:
  -d, --docs <dir>         Document directory (alternative to positional arg)
  -s, --skills <dir>       Optional skills directory
      --max-file-bytes N   Limit per file size (default: 262144)
      --max-total-bytes N  Limit total bytes in prompt (default: 2097152)
      --disable-global-skills Disable Codex CLI global skills feature
  -q, --quiet              Disable progress output (show only final response)
  -v, --verbose            Print debug logs from Codex CLI execution
  -h, --help               Show help

Examples:
  codex-orchestrator ./docs
  codex-orchestrator -d ./docs -s ./skills
  codex-orchestrator -d ./docs --quiet
`
}

func joinErrors(errors []error) string {
	var msgs []string
	for _, err := range errors {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "\n")
}

func fmtError(err error) string {
	if err == nil {
		return ""
	}
	return err.Error() + "\n"
}
