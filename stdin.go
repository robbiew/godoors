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

// pumpReader runs stdin reads on a helper goroutine so a read can give up
// at a deadline (terminal probes) instead of blocking forever. Reads are
// demand-driven: the goroutine reads stdin only when a caller asks for
// input, so nothing is read ahead and keys are never taken from a program
// that uses stdin after the door. The one exception is a read that timed
// out: it stays outstanding, and whatever it returns is handed to the next
// caller rather than lost.
type pumpReader struct {
	src      io.Reader
	once     sync.Once
	reqs     chan int // a caller wants up to n bytes
	results  chan pumpResult
	inflight bool  // a request is outstanding (its reader timed out)
	err      error // sticky: stdin has failed or closed
	pending  []byte

	mu       sync.Mutex
	deadline time.Time // zero: reads block until input arrives
}

type pumpResult struct {
	b   []byte
	err error
}

func newPump(src io.Reader) *pumpReader { return &pumpReader{src: src} }

func (p *pumpReader) start() {
	p.once.Do(func() {
		p.reqs = make(chan int)
		p.results = make(chan pumpResult, 1)
		go func() {
			for n := range p.reqs {
				buf := make([]byte, n)
				k, err := p.src.Read(buf)
				p.results <- pumpResult{buf[:k], err}
				if err != nil {
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
		if p.err != nil {
			return 0, p.err
		}
		p.start()
		if !p.inflight {
			p.reqs <- len(b)
			p.inflight = true
		}
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
		case res := <-p.results:
			p.inflight = false
			p.pending = res.b
			if res.err != nil {
				p.err = res.err
				if len(res.b) == 0 {
					return 0, res.err
				}
			}
		case <-timeout:
			return 0, errProbeTimeout
		}
	}
	n := copy(b, p.pending)
	p.pending = p.pending[n:]
	return n, nil
}
