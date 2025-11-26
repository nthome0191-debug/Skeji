package handlers

import (
	"encoding/json"
	"net/http"
	"skeji/internal/maestro/service"
	"skeji/pkg/logger"
)

type FlowHandler struct {
	service *service.MaestroService
	log     *logger.Logger
}

func NewFlowHandler(service *service.MaestroService, log *logger.Logger) *FlowHandler {
	return &FlowHandler{
		service: service,
		log:     log,
	}
}

type ExecuteFlowRequest struct {
	Flow  string         `json:"flow"`
	Input map[string]any `json:"input"`
}

type ExecuteFlowResponse struct {
	Success bool           `json:"success"`
	Output  map[string]any `json:"output,omitempty"`
	Error   string         `json:"error,omitempty"`
}

type ListFlowsResponse struct {
	Flows []string `json:"flows"`
}

func (h *FlowHandler) ExecuteFlow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req ExecuteFlowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("failed to decode request", "error", err)
		h.writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	if req.Flow == "" {
		h.writeError(w, http.StatusBadRequest, "flow name is required")
		return
	}

	if req.Input == nil {
		req.Input = make(map[string]any)
	}

	h.log.Info("executing flow", "flow", req.Flow)

	output, err := h.service.ExecuteFlow(req.Flow, req.Input)
	if err != nil {
		h.log.Error("flow execution failed", "flow", req.Flow, "error", err)
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := ExecuteFlowResponse{
		Success: true,
		Output:  output,
	}

	h.writeJSON(w, http.StatusOK, resp)
}

func (h *FlowHandler) ListFlows(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	flows := h.service.GetAvailableFlows()

	resp := ListFlowsResponse{
		Flows: flows,
	}

	h.writeJSON(w, http.StatusOK, resp)
}

func (h *FlowHandler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.log.Error("failed to encode response", "error", err)
	}
}

func (h *FlowHandler) writeError(w http.ResponseWriter, status int, message string) {
	resp := ExecuteFlowResponse{
		Success: false,
		Error:   message,
	}
	h.writeJSON(w, status, resp)
}
