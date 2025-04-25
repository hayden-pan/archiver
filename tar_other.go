//go:build !windows

package archiver

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// UnarchiveContext unpacks the .tar file at source to destination.
// Destination will be treated as a folder name.
func (t *Tar) UnarchiveContext(ctx context.Context, source, destination string) error {
	if !fileExists(destination) && t.MkdirAll {
		err := mkdir(destination, 0755)
		if err != nil {
			return fmt.Errorf("preparing destination: %v", err)
		}
	}

	// if the files in the archive do not all share a common
	// root, then make sure we extract to a single subfolder
	// rather than potentially littering the destination...
	if t.ImplicitTopLevelFolder {
		var err error
		destination, err = t.addTopLevelFolder(source, destination)
		if err != nil {
			return fmt.Errorf("scanning source archive: %v", err)
		}
	}

	file, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("opening source archive: %v", err)
	}
	defer file.Close()

	err = t.Open(file, 0)
	if err != nil {
		return fmt.Errorf("opening tar archive for reading: %v", err)
	}
	defer t.Close()

	for {
		if ctx.Err() != nil {
			return fmt.Errorf("context cancelled before finishing: %w", context.Cause(ctx))
		}

		err := t.untarNext(destination)
		if err == io.EOF {
			break
		}
		if err != nil {
			if t.ContinueOnError || IsIllegalPathError(err) {
				log.Printf("[ERROR] Reading file in tar archive: %v", err)
				continue
			}
			return fmt.Errorf("reading file in tar archive: %v", err)
		}
	}

	return nil
}

// addTopLevelFolder scans the files contained inside
// the tarball named sourceArchive and returns a modified
// destination if all the files do not share the same
// top-level folder.
func (t *Tar) addTopLevelFolder(sourceArchive, destination string) (string, error) {
	file, err := os.Open(sourceArchive)
	if err != nil {
		return "", fmt.Errorf("opening source archive: %v", err)
	}
	defer file.Close()

	// if the reader is to be wrapped, ensure we do that now
	// or we will not be able to read the archive successfully
	reader := io.Reader(file)
	if t.readerWrapFn != nil {
		reader, err = t.readerWrapFn(reader)
		if err != nil {
			return "", fmt.Errorf("wrapping reader: %v", err)
		}
	}
	if t.cleanupWrapFn != nil {
		defer t.cleanupWrapFn()
	}

	tr := tar.NewReader(reader)

	var files []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("scanning tarball's file listing: %v", err)
		}
		files = append(files, hdr.Name)
	}

	if multipleTopLevels(files) {
		destination = filepath.Join(destination, folderNameFromFileName(sourceArchive))
	}

	return destination, nil
}

func (t *Tar) untarNext(destination string) error {
	f, err := t.Read()
	if err != nil {
		return err // don't wrap error; calling loop must break on io.EOF
	}
	defer f.Close()

	header, ok := f.Header.(*tar.Header)
	if !ok {
		return fmt.Errorf("expected header to be *tar.Header but was %T", f.Header)
	}

	errPath := t.CheckPath(destination, header.Name)
	if errPath != nil {
		return fmt.Errorf("checking path traversal attempt: %v", errPath)
	}

	if t.StripComponents > 0 {
		if strings.Count(header.Name, "/") < t.StripComponents {
			return nil // skip path with fewer components
		}

		for i := 0; i < t.StripComponents; i++ {
			slash := strings.Index(header.Name, "/")
			header.Name = header.Name[slash+1:]
		}
	}
	return t.untarFile(f, destination, header)
}
