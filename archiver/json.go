package archiver

import (
	"io"
)

type GabeDecoder struct {
	reader io.Reader
}

func NewGabeDecoder(r io.Reader) *GabeDecoder {
	gd := &GabeDecoder{}
	gd.reader = r
	return gd
}

func (gd *GabeDecoder) Decode(tsm *TieredSmapMessage) {
}
