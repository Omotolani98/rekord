package ptyx

import "os"

type Options struct {
	Command []string
	Shell   string
	CWD     string
	Env     []string
	Cols    int
	Rows    int
}

type PTY interface {
	Read(p []byte) (int, error)
	Write(p []byte) (int, error)
	Resize(cols, rows int) error
	Signal(sig os.Signal) error
	Kill() error
	Wait() (int, error)
	Close() error
}

func (o Options) argv() (string, []string) {
	if len(o.Command) > 0 {
		return o.Command[0], o.Command[1:]
	}
	return o.Shell, nil
}
