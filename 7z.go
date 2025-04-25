package archiver

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync/atomic"

	"github.com/hayden-pan/archiver/v3/sevenzip"
)

type SevenZip struct {
	// Whether to skip extracting of existing files.
	SkipExistingFiles bool

	// The password to open archives (optional).
	Password string

	used atomic.Bool
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
	if s.used.Swap(true) {
		// Forbidden reuse of the instance to preevent multiple extractions
		return fmt.Errorf("instance already used")
	}

	sz := sevenzip.NewSevenZip(sevenzip.SevenZipOptions{SkipExistingFiles: s.SkipExistingFiles, Password: s.Password})
	err := sz.Extract(ctx, source, destination)
	if ctx.Err() != nil {
		return fmt.Errorf("context canceled before extrating done: %w, output: %v", context.Cause(ctx), err)
	}
	if err != nil {
		return err
	}
	return nil
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
