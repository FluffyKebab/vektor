package disk

import (
	"bufio"
	"context"
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
var _ vektor.Database = &Store{}

func NewStore(
	ctx context.Context,
	graphMutationPath string,
	vectorStorePath string,
	vectorSize int,
	ramStorage vektor.Database,
	vectorStore vektor.VectorStore,
	mutationHandler vektor.GraphMutater,
) (*Store, error) {
	graphDataFile, err := os.OpenFile(graphMutationPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	vectorDataFile, err := os.OpenFile(vectorStorePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	s := &Store{
		ramStorage:          ramStorage,
		vectorSize:          vectorSize,
		graphMutationWriter: graphDataFile,
		vectorWriter:        vectorDataFile,
	}

	go s.handleClose(ctx)

	err = s.readVectorsIntoStore(vectorDataFile, vectorStore)
	if err != nil {
		return s, err
	}

	err = s.readMutations(graphDataFile, mutationHandler)
	if err != nil {
		return s, err
	}

	return s, nil
}

func (s *Store) Insert(vec vektor.Vector) (vektor.Vector, error) {
	vecData, err := vec.Read()
	if err != nil {
		return nil, fmt.Errorf("reading vector data: %w", err)
	}
	if len(vecData) != s.vectorSize {
		return nil, fmt.Errorf("new vector is of size %v, must be of size %v", len(vecData), s.vectorSize)
	}

	err = s.writeVectorToDisk(vecData)
	if err != nil {
		return nil, fmt.Errorf("writing vector to disk: %w", err)
	}

	resultVec, err := s.ramStorage.Insert(vec)
	if err != nil {
		return nil, fmt.Errorf("writing vector to ram storage: %w", err)
	}

	return resultVec, nil
}

func (s *Store) writeVectorToDisk(vec []float64) error {
	return binary.Write(s.vectorWriter, binary.LittleEndian, vec)
}

func (s *Store) FindClosest(vec vektor.Vector, n int) ([]vektor.Vector, error) {
	return s.ramStorage.FindClosest(vec, n)
}

func (s *Store) OnEdgeAdded(fromID uint32, toID uint32, dist float32) { // TODO: handle errors
	s.graphMutationWriter.Write(graphMutationAction{actionAddEdge, fromID, toID, dist}.toBytes())
}

func (s *Store) OnEdgeRemoved(fromID uint32, toID uint32) {
	s.graphMutationWriter.Write(graphMutationAction{actionRemoveEdge, fromID, toID, 0}.toBytes())
}

func (s *Store) OnNodeAdded(nodeID uint32) {
	s.graphMutationWriter.Write(graphMutationAction{actionAddNode, nodeID, 0, 0}.toBytes())
}

func (s *Store) readVectorsIntoStore(f io.Reader, store vektor.VectorStore) error {
	reader := bufio.NewReaderSize(f, 1024*1024)

	for {
		vec := make([]float64, s.vectorSize)
		err := binary.Read(reader, binary.LittleEndian, &vec)
		if err != nil {
			if err != io.EOF {
				return err
			}

			break
		}

		store.Store(vec)
	}

	return nil
}

func (s *Store) readMutations(f io.Reader, mutationHandler vektor.GraphMutater) error {
	bufReader := bufio.NewReader(f)
	actionBytes := make([]byte, actionByteSize)

	for {
		_, err := io.ReadFull(bufReader, actionBytes)
		if err != nil {
			if err == io.EOF {
				break
			}

			if err == io.ErrUnexpectedEOF {
				return fmt.Errorf("%w: partial action at end of file", errMalformedMutationData)
			}

			return fmt.Errorf("reading mutation file: %w", err)
		}

		a := readActionFromBytes(actionBytes)
		switch a.action {
		case actionNone:
			break
		case actionAddNode:
			mutationHandler.AddNode(a.arg1)
		case actionAddEdge:
			mutationHandler.AddEdge(a.arg1, a.arg2, a.dist)
		case actionRemoveEdge:
			mutationHandler.RemoveEdge(a.arg1, a.arg2)
		default:
			return fmt.Errorf("%w: unknown action id %v", errMalformedMutationData, a.action)
		}
	}

	return nil
}

func (s *Store) handleClose(ctx context.Context) {
	<-ctx.Done()

	s.graphMutationWriter.Close()
	s.vectorWriter.Close()
}
