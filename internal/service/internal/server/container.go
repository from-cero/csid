package server

import (
	"net/http"
)

// Services contains all the services of the application.
// It is used to inject dependencies into handlers and servers.
type Services struct {
	// * add your services here
}

func NewServices() *Services {
	return &Services{
		// * initialize or add your services here
	}
}

// ---------------------------------------------------------------------

// Handlers contains all the handlers of the application.
// It is used to inject dependencies into servers and routes.
type Handlers struct {
	metrics http.Handler

	// * add your handlers here
}

func NewHandlers(metrics http.Handler, ss *Services) *Handlers {
	return &Handlers{
		metrics: metrics,

		// * initialize or add your handlers here
	}
}
