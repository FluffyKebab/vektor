package vektor

import (
	"fmt"
	"math"
)

type Distance func(v1 []float32, v2 []float32) (float32, error)

func Euclidean(v1 []float32, v2 []float32) (float32, error) {
	if len(v1) != len(v2) {
		return 0, fmt.Errorf("vectors of different length")
	}

	_ = v1[len(v2)-1] // Bounds Check Elimination hint

	var sum float32
	i := 0

	for ; i <= len(v1)-4; i += 4 {
		diff0 := v1[i] - v2[i]
		diff1 := v1[i+1] - v2[i+1]
		diff2 := v1[i+2] - v2[i+2]
		diff3 := v1[i+3] - v2[i+3]

		sum += (diff0 * diff0) + (diff1 * diff1) + (diff2 * diff2) + (diff3 * diff3)
	}

	for ; i < len(v1); i++ {
		diff := v1[i] - v2[i]
		sum += diff * diff
	}

	return float32(math.Sqrt(float64(sum))), nil
}
