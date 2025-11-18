package client

import (
	"context"
	"skeji/pkg/logger"
	"time"
)

type Client struct {
	Mongo *MongoClient
}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) SetMongo(log *logger.Logger, mongoURI string, mongoConnTimeout time.Duration) {
	c.Mongo = NewCMongolient(log, mongoURI, mongoConnTimeout)
}

func (c *Client) GracefulShutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if c.Mongo != nil {
		c.Mongo.GracefulShutdown(ctx)
	}
}
