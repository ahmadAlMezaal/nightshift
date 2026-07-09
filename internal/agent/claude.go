package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
)

// claudeBackend runs Anthropic's Claude Code CLI (`claude`) in print mode — Noctra's default backend.
type claudeBackend struct{}

func (claudeBackend) Name() string     { return "claude" }
func (claudeBackend) Label() string    { return "Claude Code" }
func (claudeBackend) CLI() string      { return "claude" }
func (claudeBackend) CoAuthor() string { return "Claude <noreply@anthropic.com>" }

// Run invokes `claude --print --output-format json`, unwrapping the JSON result into the log and returning usage/cost from the envelope; falls back to raw output when stdout isn't a JSON result object. When opts.MaxTokens > 0 it switches to a streaming path that aborts mid-flight once cumulative usage crosses the ceiling.
func (b claudeBackend) Run(ctx context.Context, opts RunOptions) (Usage, error) {
	var env []string
	if opts.UseAgentTeams {
		env = append(os.Environ(), "CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1")
	}

	if opts.MaxTokens > 0 {
		return b.runCapped(ctx, opts, env)
	}

	stdout, stderr, err := runCLICapture(ctx, b.CLI(), claudeArgs(opts), env, opts)

	if usage, result, ok := ParseClaudeJSON(stdout); ok {
		writeRunLog(ctx, opts, result+stderr)
		return usage, err
	}
	writeRunLog(ctx, opts, stdout+stderr)
	return ParseUsage(stdout + "\n" + stderr), err
}

// claudeArgs builds the argv for a Claude Code run (split out so the flag set is unit-testable).
func claudeArgs(opts RunOptions) []string {
	return []string{
		"--dangerously-skip-permissions",
		"--print",
		"--output-format", "json",
		"-p", opts.Prompt,
	}
}

// claudeStreamArgs mirrors claudeArgs but streams NDJSON events so cumulative usage can be watched mid-flight (stream-json requires --verbose).
func claudeStreamArgs(opts RunOptions) []string {
	return []string{
		"--dangerously-skip-permissions",
		"--print",
		"--output-format", "stream-json",
		"--verbose",
		"-p", opts.Prompt,
	}
}

// claudeStreamEvent is the subset of Claude Code's stream-json events Noctra reads: per-turn usage on assistant events (for the running total) and the final result envelope (authoritative usage/cost + summary text).
type claudeStreamEvent struct {
	Type         string      `json:"type"`
	Result       string      `json:"result"`
	TotalCostUSD float64     `json:"total_cost_usd"`
	Usage        claudeUsage `json:"usage"`
	Message      struct {
		Usage claudeUsage `json:"usage"`
	} `json:"message"`
}

type claudeUsage struct {
	InputTokens              int64 `json:"input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
}

func (u claudeUsage) total() int64 {
	return u.InputTokens + u.OutputTokens + u.CacheCreationInputTokens + u.CacheReadInputTokens
}

// runCapped streams Claude Code's NDJSON output, accumulating per-turn token usage, and cancels the run once it crosses opts.MaxTokens. The running total (summed across assistant turns) is approximate — it's a monotonic ceiling trigger, not exact accounting; the returned Usage is the authoritative final envelope when the run completes.
func (b claudeBackend) runCapped(ctx context.Context, opts RunOptions, env []string) (Usage, error) {
	runCtx, cancel := context.WithCancel(ctx)
	if opts.Timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, opts.Timeout)
	}
	defer cancel()

	cmd := exec.CommandContext(runCtx, b.CLI(), claudeStreamArgs(opts)...)
	cmd.Dir = opts.Workdir
	if env != nil {
		cmd.Env = env
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return Usage{}, fmt.Errorf("stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return Usage{}, fmt.Errorf("start claude: %w", err)
	}

	var (
		final     Usage
		result    string
		cumTokens int64
		aborted   bool
	)
	reader := bufio.NewReader(pipe)
	for {
		line, rerr := reader.ReadBytes('\n')
		if len(bytes.TrimSpace(line)) > 0 {
			var ev claudeStreamEvent
			if json.Unmarshal(bytes.TrimSpace(line), &ev) == nil {
				switch ev.Type {
				case "assistant":
					cumTokens += ev.Message.Usage.total()
					if !aborted && cumTokens >= opts.MaxTokens {
						aborted = true
						cancel()
					}
				case "result":
					in := ev.Usage.InputTokens + ev.Usage.CacheCreationInputTokens + ev.Usage.CacheReadInputTokens
					final = Usage{
						InputTokens:  in,
						OutputTokens: ev.Usage.OutputTokens,
						TotalTokens:  in + ev.Usage.OutputTokens,
						CostUSD:      ev.TotalCostUSD,
					}
					result = ev.Result
				}
			}
		}
		if rerr != nil {
			break
		}
	}
	waitErr := cmd.Wait()

	body := result
	if aborted {
		body += fmt.Sprintf("\n\n[noctra] run aborted: per-run token ceiling %d reached (cumulative ~%d)", opts.MaxTokens, cumTokens)
	}
	writeRunLog(ctx, opts, body+stderr.String())

	if aborted {
		if final.TotalTokens == 0 {
			final.TotalTokens = cumTokens
		}
		return final, fmt.Errorf("%w: cumulative ~%d tokens (ceiling %d)", ErrTokenCapExceeded, cumTokens, opts.MaxTokens)
	}
	if waitErr != nil {
		if runCtx.Err() == context.DeadlineExceeded {
			return final, fmt.Errorf("%w: %w", ErrTimedOut, waitErr)
		}
		return final, waitErr
	}
	return final, nil
}

// claudeRateLimitRe matches the usage / rate-limit markers Claude Code emits.
var claudeRateLimitRe = regexp.MustCompile(`(?i)rate.limit|usage.limit|exceeded.*limit|too many requests`)

func (claudeBackend) HasRateLimit(output string) bool {
	return claudeRateLimitRe.MatchString(output)
}
