package businessunits

import (
	"skeji/pkg/config"
	"testing"
)

const ServiceName = "business-units-integration-tests"

var cfg *config.Config

func TestMain(t *testing.T) {
	setup()
	testGet(t)
	testPost(t)
	testUpdate(t)
	testDelete(t)
	teardown()
}

func setup() {
	cfg = config.Load(ServiceName)
}

func teardown() {
	cfg.GracefulShutdown()
}

func testGet(t *testing.T) {
	testGetByIdEmptyTable(t)
	testGetBySearchEmptyTable(t)
	//...
}

func testPost(t *testing.T) {}

func testUpdate(t *testing.T) {}

func testDelete(t *testing.T) {}

func testGetByIdEmptyTable(t *testing.T) {}

func testGetBySearchEmptyTable(t *testing.T) {}

func testGetAllPaginatedEmptyTable(t *testing.T) {}

func testGetValidIdExistingRecord(t *testing.T) {}

func testGetInvalidIdExistingRecord(t *testing.T) {}

func testGetValidSearchExistingRecords(t *testing.T) {}

func testGetInvalidSearchExistingRecords(t *testing.T) {}

func testGetValidPaginationExistingRecords(t *testing.T) {}

func testGetInvalidPaginationExistingRecords(t *testing.T) {}

func testDeletedRecord(t *testing.T) {}

func testPostInvalidRecord(t *testing.T) {}

func testPostValidRecord(t *testing.T) {}

func testPostDuplicRecord(t *testing.T) {}

func testPostWithExtraJsonKeys(t *testing.T) {}

func testPostWithMissingRelevantKeys(t *testing.T) {}

func testUpdateNonExistingRecord(t *testing.T) {}
func testUpdateWithInvalidId(t *testing.T)     {}

func testUpdateDeletedRecord(t *testing.T) {}

func testUpdateWithBadFormatKeys(t *testing.T) {}

func testUpdateWithEmptyJson(t *testing.T) {}

func testDeleteNonExistingRecord(t *testing.T) {}

func testDeleteWithInvalidId(t *testing.T) {}
