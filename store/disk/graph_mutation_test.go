package disk

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGraphMutationAction_ToBytes(t *testing.T) {
	action := newGraphMutationAction(actionAddEdge, 42, 100)
	bytes := action.toBytes()

	require.Len(t, bytes, actionByteSize)

	// actionAddEdge is 1, so the first 4 bytes should represent uint32(1) in LittleEndian
	assert.Equal(t, []byte{1, 0, 0, 0}, bytes[0:4])

	// arg1 is 42
	assert.Equal(t, []byte{42, 0, 0, 0}, bytes[4:8])

	// arg2 is 100
	assert.Equal(t, []byte{100, 0, 0, 0}, bytes[8:12])
}
