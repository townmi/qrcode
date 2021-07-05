package reedsolomon

// Addition, subtraction, multiplication, and division in GF(2^8).
// Operations are performed modulo x^8 + x^4 + x^3 + x^2 + 1.

// http://en.wikipedia.org/wiki/Finite_field_arithmetic

import "log"

const (
	gfZero = gfElement(0)
	gfOne  = gfElement(1)
)

var (
	gfExpTable = [256]gfElement{
		/*   0 -   9 */ 1, 2, 4, 8, 16, 32, 64, 128, 29, 58,
		/*  10 -  19 */ 116, 232, 205, 135, 19, 38, 76, 152, 45, 90,
		/*  20 -  29 */ 180, 117, 234, 201, 143, 3, 6, 12, 24, 48,
		/*  30 -  39 */ 96, 192, 157, 39, 78, 156, 37, 74, 148, 53,
		/*  40 -  49 */ 106, 212, 181, 119, 238, 193, 159, 35, 70, 140,
		/*  50 -  59 */ 5, 10, 20, 40, 80, 160, 93, 186, 105, 210,
		/*  60 -  69 */ 185, 111, 222, 161, 95, 190, 97, 194, 153, 47,
		/*  70 -  79 */ 94, 188, 101, 202, 137, 15, 30, 60, 120, 240,
		/*  80 -  89 */ 253, 231, 211, 187, 107, 214, 177, 127, 254, 225,
		/*  90 -  99 */ 223, 163, 91, 182, 113, 226, 217, 175, 67, 134,
		/* 100 - 109 */ 17, 34, 68, 136, 13, 26, 52, 104, 208, 189,
		/* 110 - 119 */ 103, 206, 129, 31, 62, 124, 248, 237, 199, 147,
		/* 120 - 129 */ 59, 118, 236, 197, 151, 51, 102, 204, 133, 23,
		/* 130 - 139 */ 46, 92, 184, 109, 218, 169, 79, 158, 33, 66,
		/* 140 - 149 */ 132, 21, 42, 84, 168, 77, 154, 41, 82, 164,
		/* 150 - 159 */ 85, 170, 73, 146, 57, 114, 228, 213, 183, 115,
		/* 160 - 169 */ 230, 209, 191, 99, 198, 145, 63, 126, 252, 229,
		/* 170 - 179 */ 215, 179, 123, 246, 241, 255, 227, 219, 171, 75,
		/* 180 - 189 */ 150, 49, 98, 196, 149, 55, 110, 220, 165, 87,
		/* 190 - 199 */ 174, 65, 130, 25, 50, 100, 200, 141, 7, 14,
		/* 200 - 209 */ 28, 56, 112, 224, 221, 167, 83, 166, 81, 162,
		/* 210 - 219 */ 89, 178, 121, 242, 249, 239, 195, 155, 43, 86,
		/* 220 - 229 */ 172, 69, 138, 9, 18, 36, 72, 144, 61, 122,
		/* 230 - 239 */ 244, 245, 247, 243, 251, 235, 203, 139, 11, 22,
		/* 240 - 249 */ 44, 88, 176, 125, 250, 233, 207, 131, 27, 54,
		/* 250 - 255 */ 108, 216, 173, 71, 142, 1}

	gfLogTable = [256]int{
		/*   0 -   9 */ -1, 0, 1, 25, 2, 50, 26, 198, 3, 223,
		/*  10 -  19 */ 51, 238, 27, 104, 199, 75, 4, 100, 224, 14,
		/*  20 -  29 */ 52, 141, 239, 129, 28, 193, 105, 248, 200, 8,
		/*  30 -  39 */ 76, 113, 5, 138, 101, 47, 225, 36, 15, 33,
		/*  40 -  49 */ 53, 147, 142, 218, 240, 18, 130, 69, 29, 181,
		/*  50 -  59 */ 194, 125, 106, 39, 249, 185, 201, 154, 9, 120,
		/*  60 -  69 */ 77, 228, 114, 166, 6, 191, 139, 98, 102, 221,
		/*  70 -  79 */ 48, 253, 226, 152, 37, 179, 16, 145, 34, 136,
		/*  80 -  89 */ 54, 208, 148, 206, 143, 150, 219, 189, 241, 210,
		/*  90 -  99 */ 19, 92, 131, 56, 70, 64, 30, 66, 182, 163,
		/* 100 - 109 */ 195, 72, 126, 110, 107, 58, 40, 84, 250, 133,
		/* 110 - 119 */ 186, 61, 202, 94, 155, 159, 10, 21, 121, 43,
		/* 120 - 129 */ 78, 212, 229, 172, 115, 243, 167, 87, 7, 112,
		/* 130 - 139 */ 192, 247, 140, 128, 99, 13, 103, 74, 222, 237,
		/* 140 - 149 */ 49, 197, 254, 24, 227, 165, 153, 119, 38, 184,
		/* 150 - 159 */ 180, 124, 17, 68, 146, 217, 35, 32, 137, 46,
		/* 160 - 169 */ 55, 63, 209, 91, 149, 188, 207, 205, 144, 135,
		/* 170 - 179 */ 151, 178, 220, 252, 190, 97, 242, 86, 211, 171,
		/* 180 - 189 */ 20, 42, 93, 158, 132, 60, 57, 83, 71, 109,
		/* 190 - 199 */ 65, 162, 31, 45, 67, 216, 183, 123, 164, 118,
		/* 200 - 209 */ 196, 23, 73, 236, 127, 12, 111, 246, 108, 161,
		/* 210 - 219 */ 59, 82, 41, 157, 85, 170, 251, 96, 134, 177,
		/* 220 - 229 */ 187, 204, 62, 90, 203, 89, 95, 176, 156, 169,
		/* 230 - 239 */ 160, 81, 11, 245, 22, 235, 122, 117, 44, 215,
		/* 240 - 249 */ 79, 174, 213, 233, 230, 231, 173, 232, 116, 214,
		/* 250 - 255 */ 244, 234, 168, 80, 88, 175}
)

// gfElement is an element in GF(2^8).
type gfElement uint8

// newGFElement creates and returns a new gfElement.
func newGFElement(data byte) gfElement {
	return gfElement(data)
}

// gfAdd returns a + b.
func gfAdd(a, b gfElement) gfElement {
	return a ^ b
}

// gfSub returns a - b.
//
// Note addition is equivalent to subtraction in GF(2).
func gfSub(a, b gfElement) gfElement {
	return a ^ b
}

// gfMultiply returns a * b.
func gfMultiply(a, b gfElement) gfElement {
	if a == gfZero || b == gfZero {
		return gfZero
	}

	return gfExpTable[(gfLogTable[a]+gfLogTable[b])%255]
}

// gfDivide returns a / b.
//
// Divide by zero results in a panic.
func gfDivide(a, b gfElement) gfElement {
	if a == gfZero {
		return gfZero
	} else if b == gfZero {
		log.Panicln("Divide by zero")
	}

	return gfMultiply(a, gfInverse(b))
}

// gfInverse returns the multiplicative inverse of a, a^-1.
//
// a * a^-1 = 1
func gfInverse(a gfElement) gfElement {
	if a == gfZero {
		log.Panicln("No multiplicative inverse of 0")
	}

	return gfExpTable[255-gfLogTable[a]]
}
