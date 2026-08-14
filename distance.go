package vektor

import (
	"fmt"
	"math"
)

type Distance func(v1 []float64, v2 []float64) (float32, error)

func Euclidean(v1 []float64, v2 []float64) (float32, error) {
	if len(v1) != len(v2) {
		return 0, fmt.Errorf("vectors of different length")
	}

	var sum float64
	for i := 0; i < len(v1); i++ {
		sum += math.Pow(v1[i]-v2[i], 2)
	}

	return float32(math.Sqrt(sum)), nil
}
