package commands

import (
	"fmt"
	"io"
	"lever/utils"
	"os"
	"strings"
	"time"

	"golang.org/x/sys/windows"
)

const progressThreshold = 1 << 20

type progressBar struct {
	label   string
	total   int64
	current int64
	last    time.Time
	output  io.Writer
	width   int
}

func newProgressBar(label string, total int64) *progressBar {
	if total < progressThreshold {
		return nil
	}
	var mode uint32
	if windows.GetConsoleMode(windows.Handle(os.Stderr.Fd()), &mode) != nil {
		return nil
	}
	return &progressBar{label: label, total: total, output: os.Stderr}
}

func (p *progressBar) Write(data []byte) (int, error) {
	p.Add(int64(len(data)))
	return len(data), nil
}

func (p *progressBar) Add(n int64) {
	if p == nil {
		return
	}
	p.current += n
	if time.Since(p.last) >= 100*time.Millisecond {
		p.draw()
		p.last = time.Now()
	}
}

func (p *progressBar) Finish() {
	if p == nil {
		return
	}
	p.draw()
	fmt.Fprintln(p.output)
}

func (p *progressBar) draw() {
	const width = 24
	fraction := min(1, float64(p.current)/float64(p.total))
	completed := int(fraction * width)
	percent := int(fraction * 100)
	line := fmt.Sprintf("%s [%s%s] %3d%% %s/%s", p.label, strings.Repeat("=", completed), strings.Repeat(" ", width-completed), percent, utils.FormatSize(p.current), utils.FormatSize(p.total))
	fmt.Fprintf(p.output, "\r%s%s", line, strings.Repeat(" ", max(0, p.width-len(line))))
	p.width = len(line)
}
