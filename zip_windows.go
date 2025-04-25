package archiver

import (
	"context"

	"github.com/hayden-pan/archiver/v3/sevenzip"
)

func (z *Zip) UnarchiveContext(ctx context.Context, source, destination string) error {
	sz := sevenzip.NewSevenZip(sevenzip.SevenZipOptions{SkipExistingFiles: !z.OverwriteExisting, Password: z.Password})
	return sz.Extract(ctx, source, destination)
}
