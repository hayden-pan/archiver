package sevenzip

import "context"

type SevenZipOptions struct {
	SkipExistingFiles bool
	Password          string
}

type SevenZip interface {
	Extract(ctx context.Context, source, destination string) error
}

func NewSevenZip(options SevenZipOptions) SevenZip {
	return newSevenZip(options)
}
