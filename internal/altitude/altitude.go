// Package altitude decodes Gillham and 25-ft ADS-B altitude encodings.
package altitude

import "fmt"

func FromQBit(n int, q bool) (int, error) {
	if q {
		ft := 25*n - 1000
		if ft < -1000 || ft > 50175 {
			return 0, fmt.Errorf("q-bit altitude %d ft out of range", ft)
		}
		return ft, nil
	}
	return Gillham(n)
}

func Gillham(n int) (int, error) {
	if n < 0 || n > 2047 {
		return 0, fmt.Errorf("gillham code %d out of range", n)
	}
	c1 := (n >> 10) & 1
	a1 := (n >> 9) & 1
	c2 := (n >> 8) & 1
	a2 := (n >> 7) & 1
	c4 := (n >> 6) & 1
	a4 := (n >> 5) & 1
	b1 := (n >> 4) & 1
	b2 := (n >> 3) & 1
	d2 := (n >> 2) & 1
	b4 := (n >> 1) & 1
	d4 := n & 1
	fiveHundred := gray5(c1, a1, c2, a2, c4, a4, b1, b2, b4)
	oneHundred := gray2(d2, d4)
	ft := fiveHundred*500 + oneHundred*100 - 1000
	if ft < -1200 || ft > 126700 {
		return 0, fmt.Errorf("decoded gillham altitude %d ft out of range", ft)
	}
	return ft, nil
}

func gray5(c1, a1, c2, a2, c4, a4, b1, b2, b4 int) int {
	bits := []int{c1, a1, c2, a2, c4, a4, b1, b2, b4}
	n := 0
	for _, b := range bits {
		n = (n << 1) | b
	}
	return grayToBinary(n)
}

func gray2(d2, d4 int) int {
	return grayToBinary((d2 << 1) | d4)
}

func grayToBinary(n int) int {
	n ^= n >> 1
	n ^= n >> 2
	n ^= n >> 4
	n ^= n >> 8
	return n
}

func ValidFeet(ft int) error {
	if ft < -2000 || ft > 60000 {
		return fmt.Errorf("altitude %d ft outside ADS-B display range", ft)
	}
	return nil
}
