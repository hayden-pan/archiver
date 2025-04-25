//go:build !windows

package sevenzip

import (
	"context"
	"errors"
)

type unimplemented struct{}

func (s *unimplemented) Extract(_ context.Context, _, _ string) error {
	return errors.New("7z extracting not implemented")
}

func newSevenZip(_ SevenZipOptions) *unimplemented {
	return &unimplemented{}
}
