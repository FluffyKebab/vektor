package store

import (
	"context"

	"github.com/FluffyKebab/vektor"
	"github.com/FluffyKebab/vektor/store/disk"
	"github.com/FluffyKebab/vektor/store/nsw"
)

const (
	initNeighbors int = 16
	maxNeighbors  int = 32
)

func New(ctx context.Context, dist vektor.Distance, vectorSize int, graphMutationPath string, vectorStorePath string) (vektor.Database, error) {
	vectorStore := vektor.NewInMemoryVectorStore()

	graphStore := nsw.New(vectorStore, dist, maxNeighbors, initNeighbors)
	diskStore, err := disk.NewStore(ctx, graphMutationPath, vectorStorePath, vectorSize, graphStore, vectorStore, graphStore)
	if err != nil {
		return nil, err
	}

	graphStore.SetGraphObserver(diskStore)
	return diskStore, nil
}
