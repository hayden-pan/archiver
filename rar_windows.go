package archiver

import (
	"context"

	"github.com/hayden-pan/archiver/v3/sevenzip"
)

func (r *Rar) UnarchiveContext(ctx context.Context, source, destination string) error {
	sz := sevenzip.NewSevenZip(sevenzip.SevenZipOptions{SkipExistingFiles: !r.OverwriteExisting, Password: r.Password})
	return sz.Extract(ctx, source, destination)
}
