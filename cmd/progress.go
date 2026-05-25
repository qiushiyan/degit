package cmd

import (
	"io"
	"os"
	"time"

	degit "github.com/qiushiyan/degit/pkg"
	"github.com/schollz/progressbar/v3"
)

// cliProgress is the CLI-side implementation of pkg.Progress. It wraps a
// schollz/progressbar bar but defers construction until Init so the bar
// receives the correct total. Write is a no-op until Init runs.
type cliProgress struct {
	desc string
	out  io.Writer
	bar  *progressbar.ProgressBar
}

// Compile-time check that we satisfy the library interface.
var _ degit.Progress = (*cliProgress)(nil)

// newCLIProgress constructs an adapter that renders to stderr.
func newCLIProgress(desc string) *cliProgress {
	return newCLIProgressTo(os.Stderr, desc)
}

// newCLIProgressTo is the testable constructor; production callers use
// newCLIProgress, which targets os.Stderr.
func newCLIProgressTo(w io.Writer, desc string) *cliProgress {
	return &cliProgress{desc: desc, out: w}
}

func (p *cliProgress) Init(total int64) {
	if p.bar == nil {
		p.bar = progressbar.NewOptions64(
			total,
			progressbar.OptionSetDescription(p.desc),
			progressbar.OptionSetWriter(p.out),
			progressbar.OptionShowBytes(true),
			progressbar.OptionThrottle(65*time.Millisecond),
			progressbar.OptionClearOnFinish(),
			progressbar.OptionSetPredictTime(false),
		)
		return
	}
	// Redirect path: download() recursed with a new Content-Length.
	p.bar.ChangeMax64(total)
}

func (p *cliProgress) Write(b []byte) (int, error) {
	if p.bar == nil {
		return len(b), nil
	}
	return p.bar.Write(b)
}

func (p *cliProgress) Finish() {
	if p.bar != nil {
		_ = p.bar.Finish()
	}
}
