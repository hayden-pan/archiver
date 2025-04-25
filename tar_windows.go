package archiver

import (
	"context"

	"github.com/hayden-pan/archiver/v3/sevenzip"
)

func (t *Tar) UnarchiveContext(ctx context.Context, source, destination string) error {
	sz := sevenzip.NewSevenZip(sevenzip.SevenZipOptions{SkipExistingFiles: !t.OverwriteExisting})
	return sz.Extract(ctx, source, destination)
}
