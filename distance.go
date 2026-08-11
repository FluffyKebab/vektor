package vektor

import (
	"fmt"
	"math"
)

type Distance func(v1 Vector, v2 Vector) (float64, error)

func Euclidean(v1 Vector, v2 Vector) (float64, error) {
	v1Data, err := v1.Read()
	if err != nil {
		return 0, err
	}

	v2Data, err := v2.Read()
	if err != nil {
		return 0, err
	}

	if len(v1Data) != len(v2Data) {
		return 0, fmt.Errorf("vectors of different length")
	}

	var sum float64
	for i := 0; i < len(v1Data); i++ {
		sum += math.Pow(v1Data[i]-v2Data[i], 2)
	}

	return math.Sqrt(sum), nil
}
