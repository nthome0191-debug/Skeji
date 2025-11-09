package common

import (
	"os"
	"skeji/pkg/config"
)

type IntegrationTestSuite struct {
	Config     *config.Config
	HTTPClient *Client
	ServiceName string
}

func NewIntegrationTestSuite(serviceName string) *IntegrationTestSuite {
	cfg := config.Load(serviceName)

	serverURL := os.Getenv("TEST_SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}

	return &IntegrationTestSuite{
		Config:     cfg,
		HTTPClient: NewClient(serverURL),
		ServiceName: serviceName,
	}
}

func (s *IntegrationTestSuite) Teardown() {
	s.Config.GracefulShutdown()
}
