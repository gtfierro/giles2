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

// So what's the plan here? We have defined top level keys, which are:
//  - uuid/UUID
//  - Readings
//  - Metadata
//  - Properties
//  - Path
func (gd *GabeDecoder) Decode(tsm *TieredSmapMessage) {
}
