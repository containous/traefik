package runtime

import (
	"context"
	"fmt"

	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/log"
)

// GetTCPRoutersByEntryPoints returns all the tcp routers by entry points name and routers name.
func (c *Configuration) GetTCPRoutersByEntryPoints(ctx context.Context, entryPoints []string) map[string]map[string]*TCPRouterInfo {
	entryPointsRouters := make(map[string]map[string]*TCPRouterInfo)

	for rtName, rt := range c.TCPRouters {
		logger := log.FromContext(log.With(ctx, log.Str(log.RouterName, rtName)))

		entryPointsCount := 0
		for _, entryPointName := range rt.EntryPoints {
			if !contains(entryPoints, entryPointName) {
				rt.AddError(fmt.Errorf("entryPoint %q doesn't exist", entryPointName), false)
				logger.WithField(log.EntryPointName, entryPointName).
					Errorf("entryPoint %q doesn't exist", entryPointName)
				continue
			}

			if _, ok := entryPointsRouters[entryPointName]; !ok {
				entryPointsRouters[entryPointName] = make(map[string]*TCPRouterInfo)
			}

			entryPointsCount++
			rt.Using = append(rt.Using, entryPointName)

			entryPointsRouters[entryPointName][rtName] = rt
		}

		if entryPointsCount == 0 {
			rt.AddError(fmt.Errorf("no valid entryPoint for this router"), true)
			logger.Error("no valid entryPoint for this router")
		}
	}

	return entryPointsRouters
}

// TCPRouterInfo holds information about a currently running TCP router.
type TCPRouterInfo struct {
	*dynamic.TCPRouter          // dynamic configuration
	Err                []string `json:"error,omitempty"` // initialization error
	// Status reports whether the router is disabled, in a warning state, or all good (enabled).
	// If not in "enabled" state, the reason for it should be in the list of Err.
	// It is the caller's responsibility to set the initial status.
	Status string   `json:"status,omitempty"`
	Using  []string `json:"using,omitempty"` // Effective entry points used by that router.
}

// AddError adds err to r.Err, if it does not already exist.
// If critical is set, r is marked as disabled.
func (r *TCPRouterInfo) AddError(err error, critical bool) {
	for _, value := range r.Err {
		if value == err.Error() {
			return
		}
	}

	r.Err = append(r.Err, err.Error())
	if critical {
		r.Status = StatusDisabled
		return
	}

	// only set it to "warning" if not already in a worse state
	if r.Status != StatusDisabled {
		r.Status = StatusWarning
	}
}

// TCPServiceInfo holds information about a currently running TCP service.
type TCPServiceInfo struct {
	*dynamic.TCPService          // dynamic configuration
	Err                 []string `json:"error,omitempty"` // initialization error
	// Status reports whether the service is disabled, in a warning state, or all good (enabled).
	// If not in "enabled" state, the reason for it should be in the list of Err.
	// It is the caller's responsibility to set the initial status.
	Status string   `json:"status,omitempty"`
	UsedBy []string `json:"usedBy,omitempty"` // list of routers using that service
}

// AddError adds err to s.Err, if it does not already exist.
// If critical is set, s is marked as disabled.
func (s *TCPServiceInfo) AddError(err error, critical bool) {
	for _, value := range s.Err {
		if value == err.Error() {
			return
		}
	}

	s.Err = append(s.Err, err.Error())
	if critical {
		s.Status = StatusDisabled
		return
	}

	// only set it to "warning" if not already in a worse state
	if s.Status != StatusDisabled {
		s.Status = StatusWarning
	}
}

// TCPMiddlewareInfo holds information about a currently running middleware.
type TCPMiddlewareInfo struct {
	*dynamic.TCPMiddleware // dynamic configuration
	// Err contains all the errors that occurred during service creation.
	Err    []string `json:"error,omitempty"`
	Status string   `json:"status,omitempty"`
	UsedBy []string `json:"usedBy,omitempty"` // list of TCP routers and services using that middleware.
}

// AddError adds err to s.Err, if it does not already exist.
// If critical is set, m is marked as disabled.
func (m *TCPMiddlewareInfo) AddError(err error, critical bool) {
	for _, value := range m.Err {
		if value == err.Error() {
			return
		}
	}

	m.Err = append(m.Err, err.Error())
	if critical {
		m.Status = StatusDisabled
		return
	}

	// only set it to "warning" if not already in a worse state
	if m.Status != StatusDisabled {
		m.Status = StatusWarning
	}
}
