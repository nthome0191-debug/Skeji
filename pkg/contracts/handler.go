package contracts

import "github.com/julienschmidt/httprouter"

type Handler interface {
	RegisterRoutes(*httprouter.Router)
}
