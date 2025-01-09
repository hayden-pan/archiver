package archiver

import (
	"bytes"
	"context"
	"fmt"
	"hash"
	"hash/crc32"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/bodgit/sevenzip"
)

type SevenZip struct {
	// Whether to overwrite existing files; if false,
	// an error is returned if the file exists.
	OverwriteExisting bool

	// If true, errors encountered during reading or writing
	// a file within an archive will be logged and the
	// operation will continue on remaining files.
	ContinueOnError bool

	// Whether to preserve the modification time when extracting files.
	PreserveModTime bool

	// The password to open archives (optional).
	Password string

	// Whether to verify the checksum of the extracted file.
	// Unrecommended for skipping checksum verification.
	SkipVerifyChecksum bool

	Concurrency int

	// The path of the extracted files. Used for checking duplicate of files.
	// [string]struct{} is used to save memory.
	extractedPaths sync.Map
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
	rc, err := sevenzip.OpenReaderWithPassword(source, s.Password)
	if err != nil {
		return err
	}
	defer rc.Close()

	if err := s.parallelExtractFiles(ctx, rc.File, destination); err != nil {
		return fmt.Errorf("parallel extract files: %w", err)
	}

	return nil
}

func (s *SevenZip) parallelExtractFiles(ctx context.Context, files []*sevenzip.File, dest string) error {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(context.Canceled)

	concurrency := newConecurrencyChan(s.Concurrency)

	var wg sync.WaitGroup

FilesLoop:
	for _, f := range files {
		select {
		case concurrency <- struct{}{}:
		case <-ctx.Done():
			break FilesLoop
		}

		wg.Add(1)
		go func(f *sevenzip.File) {
			defer func() {
				<-concurrency
				wg.Done()
			}()

			if err := s.extractFile(f, dest); err != nil {
				if s.ContinueOnError {
					log.Printf("[ERROR] Extracting file %s: %v", f.Name, err)
				} else {
					cancel(fmt.Errorf("extracting file %s: %w", f.Name, err))
				}
			}
		}(f)
	}

	wg.Wait()

	err := context.Cause(ctx)
	return err
}

func newConecurrencyChan(num int) chan struct{} {
	if num > 0 {
		return make(chan struct{}, num)
	}

	concurrency := runtime.GOMAXPROCS(0)
	if concurrency <= 0 {
		concurrency = runtime.NumCPU()
	}
	if concurrency <= 0 {
		concurrency = 1
	}
	return make(chan struct{}, concurrency)
}

func (s *SevenZip) extractFile(f *sevenzip.File, dest string) error {
	if err := s.CheckPath(dest, f.Name); err != nil {
		return fmt.Errorf("checking path traversal attempt: %v", err)
	}

	path := filepath.Join(dest, f.Name)
	if _, loaded := s.extractedPaths.LoadOrStore(path, struct{}{}); loaded {
		return fmt.Errorf("file path already present at least twice: %s", path)
	}

	if f.FileInfo().IsDir() {
		if err := mkdir(path, f.FileInfo().Mode()); err != nil {
			return err
		}
		return nil
	}

	if !s.OverwriteExisting && fileExists(path) {
		return fmt.Errorf("file already exists: %s", path)
	}

	return s.writeFile(f, path)
}

func (s *SevenZip) writeFile(f *sevenzip.File, path string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	checksum := !s.SkipVerifyChecksum

	var reader io.Reader
	var hasher hash.Hash32
	if checksum {
		hasher = crc32.NewIEEE()
		reader = io.TeeReader(rc, hasher)
	} else {
		reader = rc
	}
	if err := writeNewFile(path, reader, f.FileInfo().Mode()); err != nil {
		return err
	}
	if hasher != nil {
		if hasher.Sum32() != f.CRC32 {
			return fmt.Errorf("checksum mismatch for %s", f.Name)
		}
	}
	if s.PreserveModTime {
		mod := f.FileInfo().ModTime()
		if err := os.Chtimes(path, mod, mod); err != nil {
			return fmt.Errorf("setting modtime for %s err: %v", path, err)
		}
	}

	return nil
}

func (*SevenZip) CheckPath(to, filename string) error {
	to, _ = filepath.Abs(to)
	dest := filepath.Join(to, filename)
	//prevent path traversal attacks
	if !strings.HasPrefix(dest, to) {
		return &IllegalPathError{AbsolutePath: dest, Filename: filename}
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
