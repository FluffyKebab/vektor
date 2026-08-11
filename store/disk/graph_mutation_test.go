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

func TestReadActionsFromBytes(t *testing.T) {
	t.Run("Successfully reads multiple valid actions", func(t *testing.T) {
		a1 := newGraphMutationAction(actionAddNode, 10, 0)
		a2 := newGraphMutationAction(actionAddEdge, 10, 20)
		a3 := newGraphMutationAction(actionRemoveEdge, 10, 30)

		// Simulate reading a contiguous chunk of bytes from disk
		var diskData []byte
		diskData = append(diskData, a1.toBytes()...)
		diskData = append(diskData, a2.toBytes()...)
		diskData = append(diskData, a3.toBytes()...)

		actions, err := readActionsFromBytes(diskData)

		require.NoError(t, err)
		require.Len(t, actions, 3)

		assert.Equal(t, actionAddNode, actions[0].action)
		assert.Equal(t, uint32(10), actions[0].arg1)

		assert.Equal(t, actionAddEdge, actions[1].action)
		assert.Equal(t, uint32(20), actions[1].arg2)

		assert.Equal(t, actionRemoveEdge, actions[2].action)
	})

	t.Run("Returns early if actionNone is encountered", func(t *testing.T) {
		a1 := newGraphMutationAction(actionAddNode, 10, 0)
		a2 := newGraphMutationAction(actionNone, 0, 0)      // The terminator
		a3 := newGraphMutationAction(actionAddEdge, 10, 20) // Should be ignored

		var diskData []byte
		diskData = append(diskData, a1.toBytes()...)
		diskData = append(diskData, a2.toBytes()...)
		diskData = append(diskData, a3.toBytes()...)

		actions, err := readActionsFromBytes(diskData)

		require.NoError(t, err)
		require.Len(t, actions, 1, "Should have stopped reading at actionNone")
		assert.Equal(t, actionAddNode, actions[0].action)
	})

	t.Run("Errors on malformed byte length", func(t *testing.T) {
		a1 := newGraphMutationAction(actionAddNode, 10, 0)

		diskData := a1.toBytes()
		// Corrupt the data by cutting off the last byte
		corruptData := diskData[:len(diskData)-1]

		_, err := readActionsFromBytes(corruptData)

		require.ErrorIs(t, err, errMalformedMutationData)
	})

	t.Run("Errors on unknown action ID", func(t *testing.T) {
		// Inject a completely invalid action ID (999)
		a1 := newGraphMutationAction(actionID(999), 10, 0)

		_, err := readActionsFromBytes(a1.toBytes())

		require.ErrorIs(t, err, errMalformedMutationData)
		assert.ErrorContains(t, err, "unknown action id 999")
	})
}
