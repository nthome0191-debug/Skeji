package flows

import (
	"fmt"
	"net/http"
	maestro "skeji/internal/maestro/core"
	"skeji/pkg/model"
)

func CreateBusinessUnit(ctx *maestro.MaestroContext) error {
	name := ctx.ExtractString("name")
	if maestro.IsMissing(name) {
		return maestro.MissingParamErr("name")
	}

	adminPhone := ctx.ExtractString("admin_phone")
	if maestro.IsMissing(adminPhone) {
		return maestro.MissingParamErr("admin_phone")
	}

	cities := ctx.ExtractStringList("cities")
	if len(cities) == 0 {
		return maestro.MissingParamErr("cities")
	}

	labels := ctx.ExtractStringList("labels")
	if len(labels) == 0 {
		return maestro.MissingParamErr("labels")
	}
	businessUnit := &model.BusinessUnit{
		Name:       name,
		AdminPhone: adminPhone,
		Cities:     cities,
		Labels:     labels,
	}
	timeZone := ctx.ExtractString("time_zone")
	if !maestro.IsMissing(timeZone) {
		businessUnit.TimeZone = timeZone
	}

	websiteURLs := ctx.ExtractStringList("website_urls")
	if len(websiteURLs) > 0 {
		businessUnit.WebsiteURLs = websiteURLs
	}
	if maintainersVal, exists := ctx.Input["maintainers"]; exists && maintainersVal != nil {
		if maintainers, ok := maintainersVal.(map[string]string); ok {
			businessUnit.Maintainers = maintainers
		}
	}
	resp, err := ctx.Client.BusinessUnitClient.Create(businessUnit)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to create business unit: %+v", resp.ToString())
	}
	createdBU, err := ctx.Client.BusinessUnitClient.DecodeBusinessUnit(resp)
	if err != nil {
		return err
	}
	ctx.Output["business_unit"] = createdBU
	return nil
}
