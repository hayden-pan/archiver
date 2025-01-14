//go:build !windows

package archiver

import (
	"context"
	"errors"
)

func (s *SevenZip) extract(_ context.Context, _, _ string) error {
	return errors.New("7z extracting not implemented")
}
