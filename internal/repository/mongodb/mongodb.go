package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Repository struct {
	Client  *mongo.Client
	Context context.Context
}

func New(ctx context.Context, uri string) (*Repository, error) {
	// Check that the required arguments are not null
	if ctx == nil {
		return nil, fmt.Errorf("null pointer: context is required")
	}

	if uri == "" {
		return nil, fmt.Errorf("null pointer: uri is required")
	}

	// Connect to the mongo database specified by the uri
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("error connecting to mongo: %w", err)
	}

	// Return the new storage with the mongo client and context
	return &Repository{
		Client:  client,
		Context: ctx,
	}, nil
}
