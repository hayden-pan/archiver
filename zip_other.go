//go:build !windows

package archiver

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

// UnarchiveContext unpacks the .zip file at source to destination.
// Destination will be treated as a folder name.
func (z *Zip) UnarchiveContext(ctx context.Context, source, destination string) error {
	if z.Password != "" {
		return fmt.Errorf("password-protected zip files are not supported")
	}

	if !fileExists(destination) && z.MkdirAll {
		err := mkdir(destination, 0755)
		if err != nil {
			return fmt.Errorf("preparing destination: %v", err)
		}
	}

	file, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("opening source file: %v", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("statting source file: %v", err)
	}

	err = z.Open(file, fileInfo.Size())
	if err != nil {
		return fmt.Errorf("opening zip archive for reading: %v", err)
	}
	defer z.Close()

	if err := z.decodeFileHeader(); err != nil {
		return fmt.Errorf("decoding file header: %v", err)
	}

	// if the files in the archive do not all share a common
	// root, then make sure we extract to a single subfolder
	// rather than potentially littering the destination...
	if z.ImplicitTopLevelFolder {
		files := make([]string, len(z.zr.File))
		for i := range z.zr.File {
			files[i] = z.zr.File[i].Name
		}
		if multipleTopLevels(files) {
			destination = filepath.Join(destination, folderNameFromFileName(source))
		}
	}

	for {
		if ctx.Err() != nil {
			return fmt.Errorf("context cancelled before all files were extracted: %w", context.Cause(ctx))
		}

		err := z.extractNext(destination)
		if err == io.EOF {
			break
		}
		if err != nil {
			if z.ContinueOnError || IsIllegalPathError(err) {
				log.Printf("[ERROR] Reading file in zip archive: %v", err)
				continue
			}
			return fmt.Errorf("reading file in zip archive: %v", err)
		}
		if z.SyncProgressCallback != nil {
			z.SyncProgressCallback(len(z.zr.File), z.ridx)
		}
	}

	return nil
}
