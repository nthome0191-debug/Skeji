package client

import (
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
