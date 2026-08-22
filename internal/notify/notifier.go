package notify

import (
	"context"
	"fmt"
	"strings"
)

type Notifier interface {
	Send(ctx context.Context, message string)

	SendSync(ctx context.Context, message string) error
}

type Multi struct {
	backends []Notifier
	labels   []string
}

func NewMulti(backends []Notifier, labels []string) *Multi {
	var (
		filtered       []Notifier
		filteredLabels []string
	)
	for i, b := range backends {
		if b == nil {
			continue
		}
		filtered = append(filtered, b)
		if i < len(labels) {
			filteredLabels = append(filteredLabels, labels[i])
		}
	}
	return &Multi{backends: filtered, labels: filteredLabels}
}

func (m *Multi) Send(ctx context.Context, message string) {
	if m == nil {
		return
	}
	for _, b := range m.backends {
		b.Send(ctx, message)
	}
}

func (m *Multi) SendSync(ctx context.Context, message string) error {
	if m == nil {
		return fmt.Errorf("notifier is nil")
	}
	var firstErr error
	for _, b := range m.backends {
		if err := b.SendSync(ctx, message); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (m *Multi) Labels() []string {
	if m == nil {
		return nil
	}
	return m.labels
}

func (m *Multi) String() string {
	if m == nil || len(m.labels) == 0 {
		return "Disabled"
	}
	return strings.Join(m.labels, ", ")
}
