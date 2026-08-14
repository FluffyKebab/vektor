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

// 4 (action) + 4 (arg1) + 4 (arg2) + 4 (dist) = 16 bytes
const actionByteSize = 16

type graphMutationAction struct {
	action actionID
	arg1   uint32
	arg2   uint32
	dist   float32
}

func (a graphMutationAction) toBytes() []byte {
	result := make([]byte, actionByteSize)

	binary.LittleEndian.PutUint32(result[0:4], uint32(a.action))
	binary.LittleEndian.PutUint32(result[4:8], a.arg1)
	binary.LittleEndian.PutUint32(result[8:12], a.arg2)
	binary.LittleEndian.PutUint32(result[12:16], math.Float32bits(a.dist))

	return result
}

func readActionFromBytes(d []byte) graphMutationAction {
	return graphMutationAction{
		actionID(binary.LittleEndian.Uint32(d[0:4])),
		binary.LittleEndian.Uint32(d[4:8]),
		binary.LittleEndian.Uint32(d[8:12]),
		math.Float32frombits(binary.LittleEndian.Uint32(d[12:16])),
	}
}
