//go:build !windows

package archiver

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/nwaples/rardecode"
)

// UnarchiveContext unpacks the .rar file at source to destination.
// Destination will be treated as a folder name. It supports
// multi-volume archives.
func (r *Rar) UnarchiveContext(ctx context.Context, source, destination string) error {
	if !fileExists(destination) && r.MkdirAll {
		err := mkdir(destination, 0755)
		if err != nil {
			return fmt.Errorf("preparing destination: %v", err)
		}
	}

	// if the files in the archive do not all share a common
	// root, then make sure we extract to a single subfolder
	// rather than potentially littering the destination...
	if r.ImplicitTopLevelFolder {
		var err error
		destination, err = r.addTopLevelFolder(source, destination)
		if err != nil {
			return fmt.Errorf("scanning source archive: %v", err)
		}
	}

	err := r.OpenFile(source)
	if err != nil {
		return fmt.Errorf("opening rar archive for reading: %v", err)
	}
	defer r.Close()

	for {
		if ctx.Err() != nil {
			return fmt.Errorf("context cancelled before extraction completed: %w", context.Cause(ctx))
		}

		err := r.unrarNext(destination)
		if err == io.EOF {
			break
		}
		if err != nil {
			if r.ContinueOnError || IsIllegalPathError(err) {
				log.Printf("[ERROR] Reading file in rar archive: %v", err)
				continue
			}
			return fmt.Errorf("reading file in rar archive: %v", err)
		}
	}

	return nil
}

// addTopLevelFolder scans the files contained inside
// the tarball named sourceArchive and returns a modified
// destination if all the files do not share the same
// top-level folder.
func (r *Rar) addTopLevelFolder(sourceArchive, destination string) (string, error) {
	file, err := os.Open(sourceArchive)
	if err != nil {
		return "", fmt.Errorf("opening source archive: %v", err)
	}
	defer file.Close()

	rc, err := rardecode.NewReader(file, r.Password)
	if err != nil {
		return "", fmt.Errorf("creating archive reader: %v", err)
	}

	var files []string
	for {
		hdr, err := rc.Next()
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

func (r *Rar) unrarNext(to string) error {
	f, err := r.Read()
	if err != nil {
		return err // don't wrap error; calling loop must break on io.EOF
	}
	defer f.Close()

	header, ok := f.Header.(*rardecode.FileHeader)
	if !ok {
		return fmt.Errorf("expected header to be *rardecode.FileHeader but was %T", f.Header)
	}

	errPath := r.CheckPath(to, header.Name)
	if errPath != nil {
		return fmt.Errorf("checking path traversal attempt: %v", errPath)
	}

	if r.StripComponents > 0 {
		if strings.Count(header.Name, "/") < r.StripComponents {
			return nil // skip path with fewer components
		}

		for i := 0; i < r.StripComponents; i++ {
			slash := strings.Index(header.Name, "/")
			header.Name = header.Name[slash+1:]
		}
	}

	return r.unrarFile(f, filepath.Join(to, header.Name))
}
