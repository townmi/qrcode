package qrcode

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"log"

	bitset "github.com/townmi/qrcode/bitset"
	reedsolomon "github.com/townmi/qrcode/reedsolomon"
)

// Encode a QR Code and return a raw PNG image
func Encode(content string, level RecoveryLevel, size int) ([]byte, error) {
	var q *QRCode

	q, err := New(content, level)

	if err != nil {
		return nil, err
	}
	return q.PNG(size)
}

type QRCode struct {
	// Original content encoded
	Content string

	// QR code type
	Level         RecoveryLevel
	VersionNumber int

	// User settable drawing options
	ForegroundColor color.Color
	BackgroundColor color.Color

	// Disable the QR code border
	DisableBorder bool

	encoder *dataEncoder
	version qrCodeVersion

	data   *bitset.Bitset
	symbol *symbol
	mask   int
}

func New(content string, level RecoveryLevel) (*QRCode, error) {
	encoders := []dataEncoderType{dataEncoderType1To9, dataEncoderType10To26, dataEncoderType27To40}

	var encoder *dataEncoder
	var encoded *bitset.Bitset
	var chosenVersion *qrCodeVersion
	var err error

	for _, t := range encoders {
		encoder = newDataEncoder(t)
		encoded, err = encoder.encode([]byte(content))

		if err != nil {
			continue
		}

		chosenVersion = chooseQRCodeVersion(level, encoder, encoded.Len())

		if chosenVersion != nil {
			break
		}
	}

	if err != nil {
		return nil, err
	} else if chosenVersion == nil {
		return nil, errors.New("content too long to encode")
	}

	q := &QRCode{
		Content: content,

		Level:         level,
		VersionNumber: chosenVersion.version,

		ForegroundColor: color.Black,
		BackgroundColor: color.White,

		encoder: encoder,
		data:    encoded,
		version: *chosenVersion,
	}

	return q, nil
}

// Bitmap returns the QR Code as a 2D array of 1-bit pixels.
func (q *QRCode) Bitmap() [][]bool {
	// Build QR code
	q.encode()
	return q.symbol.bitmap()
}

func (q *QRCode) encode() {
	numTerminatorBits := q.version.numTerminatorBitsRequired(q.data.Len())

	q.addTerminatorBits(numTerminatorBits)
	q.addPadding()

	encoded := q.encodeBlocks()

	const numMasks int = 8
	penalty := 0

	for mask := 0; mask < numMasks; mask++ {
		var s *symbol
		var err error

		s, err = buildRegularSymbol(q.version, mask, encoded, !q.DisableBorder)

		if err != nil {
			log.Panic(err.Error())
		}

		numEmptyModules := s.numEmptyModules()
		if numEmptyModules != 0 {
			log.Panicf("bug: numEmptyModules is %d (expected 0) (version=%d)", numEmptyModules, q.VersionNumber)
		}

		p := s.penaltyScore()

		//log.Printf("mask=%d p=%3d p1=%3d p2=%3d p3=%3d p4=%d\n", mask, p, s.penalty1(), s.penalty2(), s.penalty3(), s.penalty4())
		if q.symbol == nil || p < penalty {
			q.symbol = s
			q.mask = mask
			penalty = p
		}
	}
}

func (q *QRCode) Image(size int) image.Image {
	// Build QR code.
	q.encode()

	// Minimum pixels (both width and height) required.
	realSize := q.symbol.size

	// Variable size support.
	if size < 0 {
		size = size * -1 * realSize
	}

	// Actual pixels available to draw the symbol. Automatically increase the
	// image size if it's not large enough.
	if size < realSize {
		size = realSize
	}

	// Output image.
	rect := image.Rectangle{Min: image.Point{}, Max: image.Point{X: size, Y: size}}
	// rect := image.Rectangle{Min: image.Point{0, 0}, Max: image.Point{size, size}}

	// Saves a few bytes to have them in this order
	p := color.Palette([]color.Color{q.BackgroundColor, q.ForegroundColor})
	img := image.NewPaletted(rect, p)
	fgClr := uint8(img.Palette.Index(q.ForegroundColor))

	// QR code bitmap.
	bitmap := q.symbol.bitmap()

	// Map each image pixel to the nearest QR code module.
	modulesPerPixel := float64(realSize) / float64(size)
	for y := 0; y < size; y++ {
		y2 := int(float64(y) * modulesPerPixel)
		for x := 0; x < size; x++ {
			x2 := int(float64(x) * modulesPerPixel)

			v := bitmap[y2][x2]

			if v {
				pos := img.PixOffset(x, y)
				img.Pix[pos] = fgClr
			}
		}
	}

	return img
}

func (q *QRCode) PNG(size int) ([]byte, error) {
	img := q.Image(size)

	encoder := png.Encoder{CompressionLevel: png.BestCompression}

	var b bytes.Buffer
	err := encoder.Encode(&b, img)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (q *QRCode) addTerminatorBits(numTerminatorBits int) {
	q.data.AppendNumBools(numTerminatorBits, false)
}

func (q *QRCode) encodeBlocks() *bitset.Bitset {
	// Split into blocks.
	type dataBlock struct {
		data          *bitset.Bitset
		ecStartOffset int
	}

	block := make([]dataBlock, q.version.numBlocks())

	start := 0
	end := 0
	blockID := 0

	for _, b := range q.version.block {
		for j := 0; j < b.numBlocks; j++ {
			start = end
			end = start + b.numDataCodewords*8

			// Apply error correction to each block.
			numErrorCodewords := b.numCodewords - b.numDataCodewords
			block[blockID].data = reedsolomon.Encode(q.data.Substr(start, end), numErrorCodewords)
			block[blockID].ecStartOffset = end - start

			blockID++
		}
	}

	// Interleave the blocks.

	result := bitset.New()

	// Combine data blocks.
	working := true
	for i := 0; working; i += 8 {
		working = false

		for j, b := range block {
			if i >= block[j].ecStartOffset {
				continue
			}

			result.Append(b.data.Substr(i, i+8))

			working = true
		}
	}

	// Combine error correction blocks.
	working = true
	for i := 0; working; i += 8 {
		working = false

		for j, b := range block {
			offset := i + block[j].ecStartOffset
			if offset >= block[j].data.Len() {
				continue
			}

			result.Append(b.data.Substr(offset, offset+8))

			working = true
		}
	}

	// Append remainder bits.
	result.AppendNumBools(q.version.numRemainderBits, false)

	return result
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func (q *QRCode) addPadding() {
	numDataBits := q.version.numDataBits()

	if q.data.Len() == numDataBits {
		return
	}

	// Pad to the nearest codeword boundary.
	q.data.AppendNumBools(q.version.numBitsToPadToCodeword(q.data.Len()), false)

	// Pad codewords 0b11101100 and 0b00010001.
	padding := [2]*bitset.Bitset{
		bitset.New(true, true, true, false, true, true, false, false),
		bitset.New(false, false, false, true, false, false, false, true),
	}

	// Insert pad codewords alternately.
	i := 0
	for numDataBits-q.data.Len() >= 8 {
		q.data.Append(padding[i])

		i = 1 - i // Alternate between 0 and 1.
	}

	if q.data.Len() != numDataBits {
		log.Panicf("BUG: got len %d, expected %d", q.data.Len(), numDataBits)
	}
}

func (q *QRCode) ToString(inverseColor bool) string {
	bits := q.Bitmap()
	var buf bytes.Buffer
	for y := range bits {
		for x := range bits[y] {
			if bits[y][x] != inverseColor {
				buf.WriteString("  ")
			} else {
				buf.WriteString("██")
			}
		}
		buf.WriteString("\n")
	}
	return buf.String()
}

func (q *QRCode) ToSmallString(inverseColor bool) string {
	bits := q.Bitmap()
	var buf bytes.Buffer
	// if there is an odd number of rows, the last one needs special treatment
	for y := 0; y < len(bits)-1; y += 2 {
		for x := range bits[y] {
			if bits[y][x] == bits[y+1][x] {
				if bits[y][x] != inverseColor {
					buf.WriteString(" ")
				} else {
					buf.WriteString("█")
				}
			} else {
				if bits[y][x] != inverseColor {
					buf.WriteString("▄")
				} else {
					buf.WriteString("▀")
				}
			}
		}
		buf.WriteString("\n")
	}
	// special treatment for the last row if odd
	if len(bits)%2 == 1 {
		y := len(bits) - 1
		for x := range bits[y] {
			if bits[y][x] != inverseColor {
				buf.WriteString(" ")
			} else {
				buf.WriteString("▀")
			}
		}
		buf.WriteString("\n")
	}
	return buf.String()
}
