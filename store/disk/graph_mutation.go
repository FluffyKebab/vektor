package disk

import (
	"encoding/binary"
	"errors"
	"math"
)

var errMalformedMutationData = errors.New("malformed mutation data")

type actionID uint32

const (
	actionNone actionID = iota
	actionAddEdge
	actionRemoveEdge
	actionAddNode
)

// 4 (action) + 4 (arg1) + 4 (arg2) + 8 (dist) = 20 bytes
const actionByteSize = 20

type graphMutationAction struct {
	action actionID
	arg1   uint32
	arg2   uint32
	dist   float64
}

func (a graphMutationAction) toBytes() []byte {
	result := make([]byte, actionByteSize)

	binary.LittleEndian.PutUint32(result[0:4], uint32(a.action))
	binary.LittleEndian.PutUint32(result[4:8], a.arg1)
	binary.LittleEndian.PutUint32(result[8:12], a.arg2)
	binary.LittleEndian.PutUint64(result[12:20], math.Float64bits(a.dist))

	return result
}

func readActionFromBytes(d []byte) graphMutationAction {
	return graphMutationAction{
		actionID(binary.LittleEndian.Uint32(d[0:4])),
		binary.LittleEndian.Uint32(d[4:8]),
		binary.LittleEndian.Uint32(d[8:12]),
		math.Float64frombits(binary.LittleEndian.Uint64(d[12:20])),
	}
}
