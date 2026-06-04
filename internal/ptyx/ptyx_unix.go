//go:build !windows

package ptyx

import (
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
)

type unixPTY struct {
	master    *os.File
	cmd       *exec.Cmd
	closeOnce sync.Once
}

func Start(o Options) (PTY, error) {
	name, args := o.argv()
	cmd := exec.Command(name, args...)
	if o.CWD != "" {
		cmd.Dir = o.CWD
	}
	if o.Env != nil {
		cmd.Env = o.Env
	}
	master, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}
	p := &unixPTY{master: master, cmd: cmd}
	if o.Cols > 0 && o.Rows > 0 {
		_ = p.Resize(o.Cols, o.Rows)
	}
	return p, nil
}

func (p *unixPTY) Read(b []byte) (int, error)  { return p.master.Read(b) }
func (p *unixPTY) Write(b []byte) (int, error) { return p.master.Write(b) }

func (p *unixPTY) Resize(cols, rows int) error {
	return pty.Setsize(p.master, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
}

func (p *unixPTY) Signal(sig os.Signal) error {
	if p.cmd.Process == nil {
		return nil
	}
	return p.cmd.Process.Signal(sig)
}

func (p *unixPTY) Kill() error {
	if p.cmd.Process == nil {
		return nil
	}
	return p.cmd.Process.Kill()
}

func (p *unixPTY) Wait() (int, error) {
	err := p.cmd.Wait()
	code := -1
	if p.cmd.ProcessState != nil {
		code = p.cmd.ProcessState.ExitCode()
	}
	return code, err
}

func (p *unixPTY) Close() error {
	var err error
	p.closeOnce.Do(func() { err = p.master.Close() })
	return err
}
