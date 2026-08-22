package pipeline

import (
	"strings"
	"testing"
)

func TestTruncateDetail(t *testing.T) {
	short := "Hit the 5M token ceiling without finishing."
	if got := truncateDetail(short); got != short {
		t.Errorf("truncateDetail(short) = %q, want it unchanged", got)
	}

	long := strings.Repeat("x", 500)
	got := truncateDetail(long)
	if len([]rune(got)) > 301 {
		t.Errorf("truncateDetail(long) length = %d, want it bounded", len([]rune(got)))
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("truncateDetail(long) = %q, want an ellipsis suffix", got)
	}
}
