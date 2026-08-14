package store

import (
	"context"

	"github.com/FluffyKebab/vektor"
	"github.com/FluffyKebab/vektor/store/disk"
	"github.com/FluffyKebab/vektor/store/nsw"
)

const (
	defaultInitNeighbors  int = 16
	defaultMaxNeighbors       = 32
	defaultSearchEf           = 100
	defaultConstructionEf     = 200
)

func New(ctx context.Context, dist vektor.Distance, vectorSize int, graphMutationPath string, vectorStorePath string) (vektor.Database, error) {
	vectorStore := vektor.NewInMemoryVectorStore(vectorSize)

	graphStore := nsw.New(vectorStore, dist, defaultMaxNeighbors, defaultInitNeighbors, defaultSearchEf, defaultConstructionEf)
	diskStore, err := disk.NewStore(ctx, graphMutationPath, vectorStorePath, vectorSize, graphStore, vectorStore, graphStore)
	if err != nil {
		return nil, err
	}

	graphStore.SetGraphObserver(diskStore)
	return diskStore, nil
}
