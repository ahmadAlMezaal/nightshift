package dashboard

import (
	"regexp"
	"strings"
)

var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(api[_-]?key|api[_-]?token|auth[_-]?token|secret[_-]?key|access[_-]?token|bearer)\s*[=:]\s*["']?([A-Za-z0-9/+=_-]{20,})["']?`),

	regexp.MustCompile(`ghp_[A-Za-z0-9]{36,}`),
	regexp.MustCompile(`gho_[A-Za-z0-9]{36,}`),
	regexp.MustCompile(`ghs_[A-Za-z0-9]{36,}`),
	regexp.MustCompile(`github_pat_[A-Za-z0-9_]{22,}`),

	regexp.MustCompile(`lin_api_[A-Za-z0-9]{30,}`),

	regexp.MustCompile(`xox[bpsar]-[A-Za-z0-9-]{10,}`),

	regexp.MustCompile(`(?i)(token|secret|key|password|credential)\s*[=:]\s*["']?([0-9a-f]{32,})["']?`),

	regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9/+=_.-]{20,}`),

	regexp.MustCompile(`sk-ant-[A-Za-z0-9-]{20,}`),

	regexp.MustCompile(`sk-[A-Za-z0-9]{20,}`),

	regexp.MustCompile(`AIza[A-Za-z0-9_-]{35}`),
}

type Redactor struct {
	literals []string
}

func NewRedactor(literals []string) *Redactor {
	var filtered []string
	for _, v := range literals {
		if len(v) >= 8 {
			filtered = append(filtered, v)
		}
	}
	return &Redactor{literals: filtered}
}

func (r *Redactor) Redact(s string) string {
	if r == nil {
		return s
	}
	for _, lit := range r.literals {
		if strings.Contains(s, lit) {
			s = strings.ReplaceAll(s, lit, "[REDACTED]")
		}
	}
	for _, pat := range secretPatterns {
		s = pat.ReplaceAllStringFunc(s, func(match string) string {
			return "[REDACTED]"
		})
	}
	return s
}
