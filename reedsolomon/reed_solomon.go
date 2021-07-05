package reedsolomon

import (
	bitset "github.com/townmi/qrcode/bitset"
	"log"
)

func Encode(data *bitset.Bitset, numECBytes int) *bitset.Bitset {
	ecpoly := newGFPolyFromData(data)
	ecpoly = gfPolyMultiply(ecpoly, newGFPolyMonomial(gfOne, numECBytes))

	generator := rsGeneratorPoly(numECBytes)

	remainder := gfPolyRemainder(ecpoly, generator)

	result := bitset.Clone(data)
	result.AppendBytes(remainder.data(numECBytes))

	return result
}

func rsGeneratorPoly(degree int) gfPoly {
	if degree < 2 {
		log.Panic("degree < 2")
	}

	generator := gfPoly{term: []gfElement{1}}

	for i := 0; i < degree; i++ {
		nextPoly := gfPoly{term: []gfElement{gfExpTable[i], 1}}
		generator = gfPolyMultiply(generator, nextPoly)
	}
	return generator
}
