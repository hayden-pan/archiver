package archiver

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync/atomic"

	_ "embed"
)

//go:embed bin/7zr.exe
var sevenZipBin []byte

type SevenZip struct {
	// Whether to skip extracting of existing files.
	SkipExistingFiles bool

	// The password to open archives (optional).
	Password string

	used atomic.Bool

	binPath string
}

// CheckExt ensures the file extension matches the format.
func (*SevenZip) CheckExt(filename string) error {
	if !strings.HasSuffix(filename, ".7z") {
		return fmt.Errorf("filename must have a .7z extension")
	}
	return nil
}

func (s *SevenZip) Unarchive(source, destination string) error {
	return s.UnarchiveContext(context.Background(), source, destination)
}

func (s *SevenZip) UnarchiveContext(ctx context.Context, source, destination string) error {
	if err := s.prepareBin(); err != nil {
		return err
	}
	defer s.cleanupBin()

	err := s.extract(ctx, source, destination)

	if ctx.Err() != nil {
		return fmt.Errorf("context canceled before extrating done: %w, output: %v", context.Cause(ctx), err)
	}
	if err != nil {
		return err
	}

	return nil
}

func (s *SevenZip) prepareBin() error {
	if s.used.Swap(true) {
		return errors.New("the instance of the 7z unarchiver has already been used, please create another instance")
	}

	bin, err := os.CreateTemp("", "*7zr.exe")
	if err != nil {
		return fmt.Errorf("failed to create temporary 7zr.exe binary file: %w", err)
	}
	defer bin.Close()
	s.binPath = bin.Name()

	if _, err := bin.Write(sevenZipBin); err != nil {
		return fmt.Errorf("failed to write 7zr.exe binary file: %w", err)
	}
	return nil
}

func (s *SevenZip) cleanupBin() {
	if s.binPath != "" {
		_ = os.Remove(s.binPath)
	}
}

func (s *SevenZip) Match(file io.ReadSeeker) (bool, error) {
	currentPos, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return false, err
	}
	_, err = file.Seek(0, 0)
	if err != nil {
		return false, err
	}
	defer func() {
		_, _ = file.Seek(currentPos, io.SeekStart)
	}()

	buf := make([]byte, 6)
	if n, err := file.Read(buf); err != nil || n < 6 {
		return false, nil
	}
	return bytes.Equal(buf, []byte("7z\xBC\xAF\x27\x1C")), nil
}

func NewSevenZip() *SevenZip {
	return &SevenZip{}
}
