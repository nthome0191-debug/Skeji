# Handler File Organization Template

This document defines the standard structure for all HTTP handler files in the Skeji project. Follow this template when creating new handlers to ensure consistency across microservices.

## File Structure

```go
package handler

// 1. IMPORTS
// Organize imports in three groups (separated by blank lines):
// - Standard library imports
// - External dependencies
// - Internal project imports
import (
	"encoding/json"      // stdlib
	"net/http"          // stdlib

	"github.com/julienschmidt/httprouter"  // external

	"skeji/internal/service"  // internal
	httputil "skeji/pkg/http" // internal
)

// 2. TYPE DEFINITIONS
// Define response structs first (if needed), then handler struct
type XxxResponse struct {
	Field string `json:"field"`
}

type XxxHandler struct {
	service service.XxxService
	log     *logger.Logger  // Always include logger for observability
}

// 3. CONSTRUCTOR
// Constructor should always be named New<HandlerName>Handler
func NewXxxHandler(service service.XxxService, log *logger.Logger) *XxxHandler {
	return &XxxHandler{
		service: service,
		log:     log,
	}
}

// 4. HTTP HANDLER METHODS
// Section comment to separate from constructor

func (h *XxxHandler) Create(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Implementation
}

func (h *XxxHandler) GetByID(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Implementation
}

// ... more handler methods

// 5. HELPER FUNCTIONS (if any)
// Section comment to separate from handler methods

func helperFunction(param string) string {
	// Implementation
}

// 6. ROUTE REGISTRATION
// Section comment - always last in file

func (h *XxxHandler) RegisterRoutes(router *httprouter.Router) {
	router.POST("/api/v1/resource", h.Create)
	router.GET("/api/v1/resource/:id", h.GetByID)
	// ... more routes
}
```

## Section Order (Required)

1. **Package declaration**
2. **Imports** (stdlib → external → internal)
3. **Type definitions** (response structs, handler struct)
4. **Constructor** (New...Handler function)
5. **HTTP handler methods** (with section comment)
6. **Helper functions** (with section comment, if any exist)
7. **Route registration** (with section comment, always last)

## Import Organization

Follow Go conventions with 3 groups:
```go
import (
	// Standard library - alphabetical
	"context"
	"encoding/json"
	"net/http"

	// External dependencies - alphabetical
	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/mongo"

	// Internal packages - alphabetical
	"skeji/internal/service"
	httputil "skeji/pkg/http"
	"skeji/pkg/logger"
)
```

## Best Practices

### 1. Always Use httputil Package
```go
// ✅ Good - use httputil
httputil.WriteJSON(w, http.StatusOK, response)
httputil.WriteError(w, err)
httputil.WriteSuccess(w, data)

// ❌ Bad - manual header setting
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
w.Write([]byte(`{"status":"ok"}`))
```

### 2. Define Response Structs
```go
// ✅ Good - typed response
type HealthResponse struct {
	Status string `json:"status"`
}
httputil.WriteJSON(w, http.StatusOK, HealthResponse{Status: "ok"})

// ❌ Bad - inline map
httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
```

### 3. Section Comments
```go
// HTTP handler methods

func (h *XxxHandler) Create(...) { ... }

// Helper functions

func splitAndTrim(...) { ... }

// Route registration

func (h *XxxHandler) RegisterRoutes(...) { ... }
```

### 4. Handler Method Order
Organize CRUD methods in logical order:
1. Create (POST)
2. GetByID (GET with ID)
3. GetAll/List (GET without ID)
4. Update (PATCH/PUT)
5. Delete (DELETE)
6. Custom operations (Search, etc.)

### 5. Always Include Logger
Every handler should accept and use a logger for observability:
```go
type XxxHandler struct {
	service service.XxxService
	log     *logger.Logger  // Required
}
```

## Examples

See reference implementations:
- `business_unit.go` - Full CRUD handler with helper functions
- `health.go` - Simple health check handler

## Notes

- This structure ensures consistency across all microservices
- Makes code reviews easier
- Improves onboarding for new developers
- Facilitates code generation and scaffolding tools
