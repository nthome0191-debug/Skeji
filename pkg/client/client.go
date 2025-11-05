package client

import (
	"context"
	"skeji/pkg/logger"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Client struct {
	Mongo *mongo.Client
}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) SetMongo(log *logger.Logger, mongoURI string, mongoConnTimeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), mongoConnTimeout)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal("Failed to connect to MongoDB",
			"error", err,
			"uri", mongoURI,
		)
	}

	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal("Failed to ping MongoDB", "error", err)
	}

	log.Info("Successfully connected to MongoDB")
	c.Mongo = client
}
