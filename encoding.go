package avsproperty

import (
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
)

type Encoding struct {
	codepage int
	name     string
	charset  encoding.Encoding
}

func (e *Encoding) String() string {
	if e == EncodingNone ||
		e == nil {
		return "None"
	}
	return e.name
}

func (e *Encoding) encoder() *encoding.Encoder {
	if e.charset == nil {
		return nil
	}
	return e.charset.NewEncoder()
}

func (e *Encoding) decoder() *encoding.Decoder {
	if e.charset == nil {
		return nil
	}
	return e.charset.NewDecoder()
}

var (
	EncodingNone = &Encoding{
		name:    "",
		charset: nil,
	}
	EncodingASCII = &Encoding{
		codepage: 1,
		name:     "ASCII",
		charset:  nil,
	}
	EncodingLatin1 = &Encoding{
		codepage: 2,
		name:     "ISO-8859-1",
		charset:  charmap.ISO8859_1,
	}
	EncodingEUCJP = &Encoding{
		codepage: 3,
		name:     "EUC-JP",
		charset:  japanese.EUCJP,
	}
	EncodingSJIS = &Encoding{
		codepage: 4,
		name:     "SHIFT_JIS",
		charset:  japanese.ShiftJIS,
	}
	EncodingUTF8 = &Encoding{
		codepage: 5,
		name:     "UTF-8",
		charset:  nil,
	}

	// order matters!
	encodingLut = []*Encoding{
		EncodingNone,
		EncodingASCII,
		EncodingLatin1,
		EncodingEUCJP,
		EncodingSJIS,
		EncodingUTF8,
	}
)

func EncodingByName(name string) *Encoding {
	switch strings.ToUpper(name) {
	case "ASCII":
		return EncodingASCII

	case "ISO_8859-1":
		fallthrough
	case "ISO-8859-1":
		return EncodingLatin1

	case "EUC-JP":
		fallthrough
	case "EUC_JP":
		fallthrough
	case "EUCJP":
		return EncodingEUCJP

	case "SHIFT_JIS":
		fallthrough
	case "SHIFT-JIS":
		fallthrough
	case "SJIS":
		return EncodingSJIS

	case "":
		fallthrough
	case "NONE":
		return EncodingNone

	case "UTF-8":
		fallthrough
	case "UTF8":
		return EncodingUTF8

	default:
		return nil
	}
}

func encodingById(id byte) *Encoding {
	if int(id) >= len(encodingLut) {
		return nil
	}
	return encodingLut[id]
}
