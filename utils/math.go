package utils

import "math/big"

func Average(values []*big.Int) *big.Int {
	total := big.NewInt(0)
	for _, v := range values {
		total.Add(total, v)
	}
	return big.NewInt(0).Div(total, big.NewInt(int64(len(values))))
}

func Sum(values []*big.Int) *big.Int {
	sum := big.NewInt(0)
	for _, v := range values {
		sum.Add(sum, v)
	}
	return sum
}
