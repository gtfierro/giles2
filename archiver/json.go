package archiver

import (
	"bufio"
	"fmt"
	"io"
)

type GabeDecoder struct {
	buf *bufio.Reader
}

func NewGabeDecoder(r io.Reader) *GabeDecoder {
	gd := &GabeDecoder{}
	gd.buf = bufio.NewReader(r)
	return gd
}

// So what's the plan here? We have defined top level keys, which are:
//  - uuid/UUID
//  - Readings
//  - Metadata
//  - Properties
//  - Path
// Start with UUID: once we find location of UUID
func (gd *GabeDecoder) Decode(tsm *TieredSmapMessage) error {
	var err error
	tsm = &TieredSmapMessage{}
	// detect start and fail early if no match
	if err = gd.expect('{'); err != nil {
		return err
	}
	if err = gd.takeSpaces(); err != nil {
		return err
	}
	if err = gd.expect('"'); err != nil {
		return err
	}
	// read until next " to get the Path
	pathBytes, err := gd.buf.ReadBytes('"')
	if err != nil {
		return err
	}
	(*tsm)[string(pathBytes[:len(pathBytes)-1])] = &SmapMessage{}
	return nil
}

// reads the next character from the buffer and checks if it matches what is expected. Returns
// an error if this is the case, else nil
func (gd *GabeDecoder) expect(want byte) error {
	if c, err := gd.buf.Peek(1); c[0] != want {
		return fmt.Errorf("Json string did not start with %q (was %q)", want, c)
	} else if err != nil {
		return err
	}
	gd.buf.Discard(1)
	return nil
}

// consumes from the underlying buffer while there is whitespace. Only consumes
// if there is whitespace at front of buffer and stops before consuming non-whitespace
func (gd *GabeDecoder) takeSpaces() error {
	var (
		c   []byte
		err error
	)

	for {
		c, err = gd.buf.Peek(1)
		if err != nil {
			return err
		}
		switch c[0] {
		case '\t', '\n', '\r', ' ':
			gd.buf.Discard(1)
		default:
			return nil
		}
	}
	return nil
}
