package main

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ADR-169: no commercial endpoint may be exposed until identity,
// authentication, permissions and audit exist (Fase 03). This test fails
// if any route other than GET /healthz is registered.
func TestOnlyHealthEndpointIsExposed(t *testing.T) {
	r, ok := router(nil).(chi.Routes)
	require.True(t, ok)
	var routes []string
	require.NoError(t, chi.Walk(r, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		routes = append(routes, method+" "+route)
		return nil
	}))
	assert.Equal(t, []string{"GET /healthz"}, routes)
}
