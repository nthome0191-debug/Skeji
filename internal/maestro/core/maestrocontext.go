package core

import (
	"fmt"
	"skeji/pkg/client"
	"skeji/pkg/logger"
	"time"
)

type MaestroContext struct {
	Input   map[string]any
	Process map[string]any
	Output  map[string]any
	Client  *client.Client
	Logger  *logger.Logger
}

func NewMaestroContext(input map[string]any, client *client.Client, logger *logger.Logger) *MaestroContext {
	return &MaestroContext{
		Input:  input,
		Output: make(map[string]any),
		Client: client,
		Logger: logger,
	}
}

func (ctx *MaestroContext) ExtractString(key string) string {
	if val, exists := ctx.Input[key]; exists {
		return val.(string)
	}
	return ""
}

func (ctx *MaestroContext) ExtractStringList(key string) []string {
	if val, exists := ctx.Input[key]; exists {
		return val.([]string)
	}
	return []string{}
}

func (ctx *MaestroContext) ExtractTime(key string) (time.Time, error) {
	val, exists := ctx.Input[key]
	if !exists || val == nil {
		return time.Time{}, fmt.Errorf("key %s not found", key)
	}

	switch v := val.(type) {

	case string:
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid datetime format for key %s: must be RFC3339 (e.g. 2025-11-19T14:00:00Z): %v", key, err)
		}
		return t, nil

	case int64:
		return time.Unix(v, 0), nil

	case int:
		return time.Unix(int64(v), 0), nil

	case float64:
		return time.Unix(int64(v), 0), nil

	case time.Time:
		return v, nil

	default:
		return time.Time{}, fmt.Errorf("unsupported datetime type for key %s: %T", key, v)
	}
}
