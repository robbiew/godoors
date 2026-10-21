package godoors

import (
	"errors"
	"io"
	"sync"
	"time"
)

// errProbeTimeout is returned by a stdin read that hit the probe deadline:
// the terminal never answered a query.
var errProbeTimeout = errors.New("godoors: no reply from terminal")

// pumpReader moves stdin reads onto one background goroutine, so a read
// can give up at a deadline (terminal probes) without leaving a second
// reader blocked on stdin to steal the caller's later keystrokes.
type pumpReader struct {
	src     io.Reader
	once    sync.Once
	chunks  chan []byte
	err     error // set before chunks is closed
	pending []byte

	mu       sync.Mutex
	deadline time.Time // zero: reads block until input arrives
}

func newPump(src io.Reader) *pumpReader { return &pumpReader{src: src} }

func (p *pumpReader) start() {
	p.once.Do(func() {
		p.chunks = make(chan []byte)
		go func() {
			defer close(p.chunks)
			for {
				buf := make([]byte, 512)
				n, err := p.src.Read(buf)
				if n > 0 {
					p.chunks <- buf[:n]
				}
				if err != nil {
					p.err = err
					return
				}
			}
		}()
	})
}

// setDeadline makes reads that would block past t fail with
// errProbeTimeout; the zero time removes the deadline.
func (p *pumpReader) setDeadline(t time.Time) {
	p.mu.Lock()
	p.deadline = t
	p.mu.Unlock()
}

func (p *pumpReader) Read(b []byte) (int, error) {
	if len(p.pending) == 0 {
		p.start()
		p.mu.Lock()
		d := p.deadline
		p.mu.Unlock()
		var timeout <-chan time.Time
		if !d.IsZero() {
			t := time.NewTimer(time.Until(d))
			defer t.Stop()
			timeout = t.C
		}
		select {
		case c, ok := <-p.chunks:
			if !ok {
				if p.err != nil {
					return 0, p.err
				}
				return 0, io.EOF
			}
			p.pending = c
		case <-timeout:
			return 0, errProbeTimeout
		}
	}
	n := copy(b, p.pending)
	p.pending = p.pending[n:]
	return n, nil
}
