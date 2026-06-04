//go:build windows

package ptyx

import (
	"os"
	"sync"

	"github.com/aymanbagabas/go-pty"
)

type windowsPTY struct {
	pty       pty.Pty
	cmd       *pty.Cmd
	closeOnce sync.Once
}

func Start(o Options) (PTY, error) {
	p, err := pty.New()
	if err != nil {
		return nil, err
	}
	name, args := o.argv()
	cmd := p.Command(name, args...)
	if o.CWD != "" {
		cmd.Dir = o.CWD
	}
	if o.Env != nil {
		cmd.Env = o.Env
	}
	if err := cmd.Start(); err != nil {
		_ = p.Close()
		return nil, err
	}
	wp := &windowsPTY{pty: p, cmd: cmd}
	if o.Cols > 0 && o.Rows > 0 {
		_ = wp.Resize(o.Cols, o.Rows)
	}
	return wp, nil
}

func (p *windowsPTY) Read(b []byte) (int, error)  { return p.pty.Read(b) }
func (p *windowsPTY) Write(b []byte) (int, error) { return p.pty.Write(b) }

func (p *windowsPTY) Resize(cols, rows int) error {
	return p.pty.Resize(cols, rows)
}

func (p *windowsPTY) Signal(_ os.Signal) error {
	return p.Kill()
}

func (p *windowsPTY) Kill() error {
	if p.cmd.Process == nil {
		return nil
	}
	return p.cmd.Process.Kill()
}

func (p *windowsPTY) Wait() (int, error) {
	err := p.cmd.Wait()
	code := -1
	if p.cmd.ProcessState != nil {
		code = p.cmd.ProcessState.ExitCode()
	}
	return code, err
}

func (p *windowsPTY) Close() error {
	var err error
	p.closeOnce.Do(func() { err = p.pty.Close() })
	return err
}
