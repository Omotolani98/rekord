//go:build windows

package recorder

import (
	"context"
	"errors"
)

type PTYRecorder struct{}

func NewPTYRecorder() *PTYRecorder { return &PTYRecorder{} }

func (r *PTYRecorder) Record(_ context.Context, _ Options) (Result, error) {
	return Result{ExitCode: -1}, errors.New("recorder: interactive recording is not supported on windows")
}
