package client

import (
	"context"
	"skeji/pkg/logger"
	"time"
)

type Client struct {
	Mongo              *MongoClient
	BusinessUnitClient *BusinessUnitClient
	ScheduleClient     *ScheduleClient
	BookingClient      *BookingClient
}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) SetMongo(log *logger.Logger, mongoURI string, mongoConnTimeout time.Duration) {
	c.Mongo = NewCMongolient(log, mongoURI, mongoConnTimeout)
}

func (c *Client) SetBusinessUnitClient(baseUrl string) {
	c.BusinessUnitClient = NewBusinessUnitClient(baseUrl)
}

func (c *Client) SetSchdeuleClient(baseUrl string) {
	c.ScheduleClient = NewScheduleClient(baseUrl)
}

func (c *Client) SetBookingClient(baseUrl string) {
	c.BookingClient = NewBookingClient(baseUrl)
}

func (c *Client) GracefulShutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if c.Mongo != nil {
		c.Mongo.GracefulShutdown(ctx)
	}
}
