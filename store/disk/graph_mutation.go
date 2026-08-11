package disk

import (
	"encoding/binary"
	"errors"
	"fmt"
)

var errMalformedMutationData = errors.New("malformed mutation data")

type actionID uint32

const (
	actionNone actionID = iota
	actionAddEdge
	actionRemoveEdge
	actionAddNode
)

const actionByteSize = 4 + 4 + 4

type graphMutationAction struct {
	action actionID
	arg1   uint32
	arg2   uint32
}

func newGraphMutationAction(a actionID, arg1 uint32, arg2 uint32) graphMutationAction {
	return graphMutationAction{
		action: a,
		arg1:   arg1,
		arg2:   arg2,
	}
}

func (a graphMutationAction) toBytes() []byte {
	result := make([]byte, actionByteSize)
	binary.LittleEndian.PutUint32(result[0:4], uint32(a.action))
	binary.LittleEndian.PutUint32(result[4:8], a.arg1)
	binary.LittleEndian.PutUint32(result[8:12], a.arg2)

	return result
}

func readActionsFromBytes(d []byte) ([]graphMutationAction, error) {
	result := make([]graphMutationAction, 0)

	for i := 0; i < len(d); i += actionByteSize {
		if i+actionByteSize > len(d) {
			return nil, errMalformedMutationData
		}

		newAction := graphMutationAction{
			actionID(binary.LittleEndian.Uint32(d[i : i+4])),
			binary.LittleEndian.Uint32(d[i+4 : i+8]),
			binary.LittleEndian.Uint32(d[i+8 : i+12]),
		}

		if newAction.action == actionNone {
			return result, nil
		}

		if newAction.action != actionAddEdge &&
			newAction.action != actionRemoveEdge &&
			newAction.action != actionAddNode {

			return nil, fmt.Errorf("%w: unknown action id %v", errMalformedMutationData, newAction.action)
		}

		result = append(result, newAction)
	}

	return result, nil
}
