package agent

import (
	"context"
	"regexp"
)

type codexBackend struct{}

func (codexBackend) Name() string     { return "codex" }
func (codexBackend) Label() string    { return "OpenAI Codex" }
func (codexBackend) CLI() string      { return "codex" }
func (codexBackend) CoAuthor() string { return "Codex <noreply@openai.com>" }

func (b codexBackend) Run(ctx context.Context, opts RunOptions) (Usage, error) {
	out, err := runCLI(ctx, b.CLI(), codexArgs(opts), nil, opts)
	return ParseUsage(out), err
}

func codexArgs(opts RunOptions) []string {
	return []string{
		"exec",
		"--dangerously-bypass-approvals-and-sandbox",
		opts.Prompt,
	}
}

var codexRateLimitRe = regexp.MustCompile(`(?i)rate.?limit|usage.?limit|quota|exceeded.*limit|too many requests`)

func (codexBackend) HasRateLimit(output string) bool {
	return codexRateLimitRe.MatchString(output)
}
