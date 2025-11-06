package businessunits

import (
	"skeji/pkg/config"
	"testing"
)

const ServiceName = "business-units-integration-tests"

var cfg *config.Config

func TestMain(t *testing.T) {
	setup()
	teardown()
}

func setup() {
	cfg = config.Load(ServiceName)
}

func teardown() {
	cfg.GracefulShutdown()
}
