package archiver

import (
	"bytes"
	"fmt"
	"hash"
	"hash/crc32"
	"io"
	"log"
	"path/filepath"
	"strings"

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

	// The password to open archives (optional).
	Password string

	// Whether to verify the checksum of the extracted file.
	// Unrecommended for skipping checksum verification.
	SkipVerifyChecksum bool

	hash hash.Hash32
}

func (s *SevenZip) Unarchive(source, destination string) error {
	rc, err := sevenzip.OpenReaderWithPassword(source, s.Password)
	if err != nil {
		return err
	}
	defer rc.Close()

	if !s.SkipVerifyChecksum {
		s.hash = crc32.NewIEEE()
	} else {
		s.hash = nil
	}

	for _, f := range rc.File {
		err := s.extractFile(f, destination)
		if err != nil {
			if s.ContinueOnError {
				log.Printf("[ERROR] Reading file in 7z archive: %v", err)
				continue
			}
			return fmt.Errorf("reading file in 7z archive: %v", err)
		}
	}

	return nil
}

func (s *SevenZip) extractFile(f *sevenzip.File, dest string) error {
	if err := s.CheckPath(dest, f.Name); err != nil {
		return fmt.Errorf("checking path traversal attempt: %v", err)
	}

	path := filepath.Join(dest, f.Name)

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

	checksum := s.hash != nil

	var reader io.Reader
	if checksum {
		s.hash.Reset()
		reader = io.TeeReader(rc, s.hash)
	} else {
		reader = rc
	}
	if err := writeNewFile(path, reader, f.FileInfo().Mode()); err != nil {
		return err
	}
	if checksum {
		if s.hash.Sum32() != f.CRC32 {
			return fmt.Errorf("checksum mismatch for %s", f.Name)
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
