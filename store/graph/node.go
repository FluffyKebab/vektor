package graph

import (
	"errors"
	"math"
)

var ErrInvalidID = errors.New("invalid node ID")

const EmptyNodeID uint32 = math.MaxUint32

type NodeDistance struct {
	NodeID uint32
	Dist   float32
}

var EmptyNodeDistance = NodeDistance{EmptyNodeID, 0}

type Neighbors []NodeDistance

func (n Neighbors) HighestDistanceIndex() int {
	if n.Len() == 0 {
		return -1
	}

	worstIdx := 0
	maxDist := n[0].Dist
	for i := 1; i < len(n) && n[i].NodeID != EmptyNodeID; i++ {
		if n[i].Dist > maxDist {
			maxDist = n[i].Dist
			worstIdx = i
		}
	}

	return worstIdx
}

func (n Neighbors) IndexOf(neighborID uint32) int {
	for i := 0; i < len(n) && n[i].NodeID != EmptyNodeID; i++ {
		if n[i].NodeID == neighborID {
			return i
		}
	}

	return -1
}

func (n Neighbors) Len() int {
	i := 0
	for i < len(n) && n[i].NodeID != EmptyNodeID {
		i++
	}
	return i
}

func (n Neighbors) Remove(i int) {
	lastIdx := n.Len() - 1
	n[i] = n[lastIdx]
	n[lastIdx] = NodeDistance{NodeID: EmptyNodeID}
}

func (n Neighbors) AddNeighbor(nd NodeDistance) (needToRemove bool) {
	place := n.Len()
	if place >= len(n) {
		return true
	}

	n[place] = nd
	if place+1 >= len(n) {
		return true
	}

	return false
}
