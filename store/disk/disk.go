package disk

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/FluffyKebab/vektor"
)

type Store struct {
	ramStorage          vektor.Database
	vectorSize          int
	graphMutationWriter io.WriteCloser
	vectorWriter        io.WriteCloser
}

var _ vektor.GraphObserver = &Store{}

func NewStore(
	graphMutationPath string,
	vectorStorePath string,
	ramStorage vektor.Database,
	vectorSize int,
) (*Store, error) {
	f, err := os.Open(graphMutationPath)
	if err != nil {
		return nil, err
	}

	return &Store{
		ramStorage:          ramStorage,
		vectorSize:          vectorSize,
		graphMutationWriter: f,
	}, nil
}

func (s *Store) Insert(vec vektor.Vector) error {
	vecData, err := vec.Read()
	if err != nil {
		return fmt.Errorf("reading vector data: %w", err)
	}
	if len(vecData) != s.vectorSize {
		return fmt.Errorf("new vector is of size %v, must be of size %v", len(vecData), s.vectorSize)
	}

	err = s.writeVectorToDisk(vec)
	if err != nil {
		return fmt.Errorf("writing vector to disk: %w", err)
	}

	err = s.ramStorage.Insert(vec)
	if err != nil {
		return fmt.Errorf("writing vector to ram storage: %w", err)
	}

	return nil
}

func (s *Store) writeVectorToDisk(vec vektor.Vector) error {
	return binary.Write(s.vectorWriter, binary.LittleEndian, vec)
}

func (s *Store) FindClosest(vec vektor.Vector, n int) ([]vektor.Vector, error) {
	return s.ramStorage.FindClosest(vec, n)
}

func (s *Store) OnEdgeAdded(fromID uint32, toID uint32) { // TODO: handle errors
	s.graphMutationWriter.Write(newGraphMutationAction(actionAddEdge, fromID, toID).toBytes())
}

func (s *Store) OnEdgeRemoved(fromID uint32, toID uint32) {
	s.graphMutationWriter.Write(newGraphMutationAction(actionAddEdge, fromID, toID).toBytes())
}

func (s *Store) OnNodeAdded(nodeID uint32) {
	s.graphMutationWriter.Write(newGraphMutationAction(actionAddEdge, nodeID, 0).toBytes())
}
