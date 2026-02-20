// Package logbuf provides a thread-safe ring buffer for log lines and an
// slog.Handler that tees output to both stderr and the ring buffer.
package logbuf

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"sync"
)

const defaultCapacity = 500

// RingBuffer is a fixed-capacity ring buffer of log lines.
type RingBuffer struct {
	mu    sync.Mutex
	lines []string
	pos   int
	full  bool
}

// NewRingBuffer returns a ring buffer that holds up to cap lines.
func NewRingBuffer(cap int) *RingBuffer {
	if cap <= 0 {
		cap = defaultCapacity
	}
	return &RingBuffer{lines: make([]string, cap)}
}

// Write appends a line to the ring buffer.
func (rb *RingBuffer) Write(line string) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.lines[rb.pos] = line
	rb.pos = (rb.pos + 1) % len(rb.lines)
	if rb.pos == 0 {
		rb.full = true
	}
}

// Lines returns all buffered lines in chronological order.
func (rb *RingBuffer) Lines() string {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	var b strings.Builder
	if rb.full {
		for i := rb.pos; i < len(rb.lines); i++ {
			b.WriteString(rb.lines[i])
			b.WriteByte('\n')
		}
	}
	for i := range rb.pos {
		b.WriteString(rb.lines[i])
		b.WriteByte('\n')
	}
	return b.String()
}

// Handler is an slog.Handler that writes to both stderr and a RingBuffer.
type Handler struct {
	inner  slog.Handler
	buffer *RingBuffer
}

// NewHandler returns a Handler that writes text-formatted logs to stderr
// and captures them in buf.
func NewHandler(buf *RingBuffer) *Handler {
	return &Handler{
		inner:  slog.NewTextHandler(os.Stderr, nil),
		buffer: buf,
	}
}

func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	// Format the line the same way the text handler would.
	var b strings.Builder
	b.WriteString(r.Time.Format("2006/01/02 15:04:05"))
	b.WriteByte(' ')
	b.WriteString(r.Level.String())
	b.WriteByte(' ')
	b.WriteString(r.Message)
	r.Attrs(func(a slog.Attr) bool {
		b.WriteByte(' ')
		b.WriteString(a.Key)
		b.WriteByte('=')
		b.WriteString(a.Value.String())
		return true
	})
	h.buffer.Write(b.String())

	// Also write to stderr via the inner handler.
	return h.inner.Handle(ctx, r)
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{inner: h.inner.WithAttrs(attrs), buffer: h.buffer}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{inner: h.inner.WithGroup(name), buffer: h.buffer}
}
