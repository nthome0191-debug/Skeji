package service

import (
	"fmt"
	maestro "skeji/internal/maestro/core"
	"skeji/internal/maestro/flows"
	"skeji/pkg/client"
	"skeji/pkg/logger"
)

type MaestroService struct {
	client *client.Client
	Logger *logger.Logger
}

func NewMaestroService(client *client.Client, logger *logger.Logger) *MaestroService {
	return &MaestroService{
		client: client,
		Logger: logger,
	}
}

type FlowHandler func(ctx *maestro.MaestroContext) error

var flowRegistry = map[string]FlowHandler{
	"create_business_unit": flows.CreateBusinessUnit,
	"create_booking":       flows.CreateBooking,
	"get_daily_schedule":   flows.GetDailySchedule,
	"search_business":      flows.SearchBusiness,
}

func (s *MaestroService) ExecuteFlow(flowName string, input map[string]any) (map[string]any, error) {
	handler, exists := flowRegistry[flowName]
	if !exists {
		return nil, fmt.Errorf("unknown flow: %s", flowName)
	}
	ctx := maestro.NewMaestroContext(input, s.client, s.Logger)
	err := handler(ctx)
	if err != nil {
		return nil, fmt.Errorf("flow execution failed: %v", err)
	}
	return ctx.Output, nil
}

func (s *MaestroService) GetAvailableFlows() []string {
	flows := make([]string, 0, len(flowRegistry))
	for flowName := range flowRegistry {
		flows = append(flows, flowName)
	}
	return flows
}
