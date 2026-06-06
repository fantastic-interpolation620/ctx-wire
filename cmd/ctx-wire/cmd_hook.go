package main

import (
	"fmt"
	"os"
	"strings"

	"ctx-wire/internal/hook"
	"ctx-wire/internal/rewrite"
)

// cmdHook runs as an agent pre-tool hook. It always exits 0 so it can never
// block the agent; a malformed payload is a silent passthrough.
func cmdHook(args []string) int {
	if isHelpArg(args) {
		printHelp(os.Stdout, helpDoc{
			usage:   []string{"ctx-wire hook <agent>"},
			summary: "Agent pre-tool hook: read a tool-call payload on stdin, emit the rewritten command.",
			notes: []string{
				"Wired up for you by `ctx-wire init <agent>`. agent is one of: claude, codex, cursor, gemini, copilot.",
			},
		})
		return 0
	}
	if len(args) == 0 {
		usageLine(os.Stderr, "ctx-wire hook <agent>")
		return 2
	}
	switch args[0] {
	case "claude":
		_ = hook.Claude(os.Stdin, os.Stdout)
	case "cursor":
		_ = hook.Cursor(os.Stdin, os.Stdout)
	case "codex":
		_ = hook.Codex(os.Stdin, os.Stdout)
	case "gemini":
		_ = hook.Gemini(os.Stdin, os.Stdout)
	case "copilot":
		_ = hook.Copilot(os.Stdin, os.Stdout)
	default:
		// Unknown agent: passthrough rather than block.
	}
	return 0
}

// cmdRewrite prints the rewritten form of a shell command line.
func cmdRewrite(args []string) int {
	if isHelpArg(args) {
		printHelp(os.Stdout, helpDoc{
			usage:   []string{"ctx-wire rewrite [--agent <agent>] <command line>"},
			summary: "Print how the hook would rewrite a shell command line (a debugging aid).",
			examples: []string{
				`ctx-wire rewrite "git status && ls -la | head"`,
				`ctx-wire rewrite --agent opencode "git status"`,
			},
			notes: []string{
				"Shows the rewrite only; it runs nothing. For the filter/mode decision, use `ctx-wire explain`.",
			},
		})
		return 0
	}
	if len(args) == 0 {
		usageLine(os.Stderr, "ctx-wire rewrite [--agent <agent>] <command line>")
		return 2
	}
	agentName := ""
	if args[0] == "--agent" {
		if len(args) < 3 {
			usageLine(os.Stderr, "ctx-wire rewrite --agent <agent> <command line>")
			return 2
		}
		agentName = args[1]
		args = args[2:]
	} else if strings.HasPrefix(args[0], "--agent=") {
		agentName = strings.TrimPrefix(args[0], "--agent=")
		args = args[1:]
		if agentName == "" || len(args) == 0 {
			usageLine(os.Stderr, "ctx-wire rewrite --agent <agent> <command line>")
			return 2
		}
	}
	line := strings.Join(args, " ")
	if agentName != "" {
		fmt.Println(rewrite.LineForAgent(line, agentName))
		return 0
	}
	fmt.Println(rewrite.Line(line))
	return 0
}
