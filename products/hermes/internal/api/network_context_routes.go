package api

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/mca-rolando/HermesDDNS/internal/networkcontext"
)

type agentNetworkContextWANRequest struct {
	InterfaceName string `json:"interface_name" validate:"required,max=128"`
	Role          string `json:"role" validate:"omitempty,oneof=primary secondary other"`
	DefaultRoute  bool   `json:"default_route"`
	IPv4          string `json:"ipv4" validate:"omitempty,max=64"`
	GatewayIPv4   string `json:"gateway_ipv4" validate:"omitempty,max=64"`
	IPv6          string `json:"ipv6" validate:"omitempty,max=128"`
	PublicIPv4    string `json:"public_ipv4" validate:"omitempty,max=64"`
}

type agentNetworkContextSegmentRequest struct {
	Name        string `json:"name" validate:"required,max=128"`
	VLANID      *int   `json:"vlan_id" validate:"omitempty,min=1,max=4094"`
	IPv4CIDR    string `json:"ipv4_cidr" validate:"omitempty,max=64"`
	GatewayIPv4 string `json:"gateway_ipv4" validate:"omitempty,max=64"`
	Purpose     string `json:"purpose" validate:"omitempty,max=64"`
}

type agentNetworkContextRequest struct {
	WANs     []agentNetworkContextWANRequest     `json:"wans" validate:"required,min=1,max=16,dive"`
	Networks []agentNetworkContextSegmentRequest `json:"networks" validate:"max=512,dive"`
}

func (s *Server) agentNetworkContext(c echo.Context) error {
	auth, ok := agentContext(c)
	if !ok || s.NetworkContext == nil {
		return agentUnauthorized(c)
	}

	var req agentNetworkContextRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	input := networkcontext.ReportInput{
		WANs:     make([]networkcontext.WANInput, 0, len(req.WANs)),
		Networks: make([]networkcontext.SegmentInput, 0, len(req.Networks)),
	}
	for _, wan := range req.WANs {
		input.WANs = append(input.WANs, networkcontext.WANInput{
			InterfaceName: wan.InterfaceName,
			Role:          wan.Role,
			DefaultRoute:  wan.DefaultRoute,
			IPv4:          wan.IPv4,
			GatewayIPv4:   wan.GatewayIPv4,
			IPv6:          wan.IPv6,
			PublicIPv4:    wan.PublicIPv4,
		})
	}
	for _, network := range req.Networks {
		input.Networks = append(input.Networks, networkcontext.SegmentInput{
			Name:        network.Name,
			VLANID:      network.VLANID,
			IPv4CIDR:    network.IPv4CIDR,
			GatewayIPv4: network.GatewayIPv4,
			Purpose:     network.Purpose,
		})
	}

	context, err := s.NetworkContext.Report(auth.Device.ID, s.callerIP(c), input)
	if err != nil {
		return s.networkContextError(c, err)
	}
	return c.JSON(http.StatusOK, context)
}

func (s *Server) getNetworkContext(c echo.Context) error {
	deviceID, err := parseUintParam(c, "id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid device id"})
	}
	if s.NetworkContext == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "network context service unavailable"})
	}
	context, err := s.NetworkContext.Context(deviceID)
	if err != nil {
		return s.networkContextError(c, err)
	}
	return c.JSON(http.StatusOK, context)
}

func (s *Server) listNetworkContexts(c echo.Context) error {
	if s.NetworkContext == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "network context service unavailable"})
	}
	contexts, err := s.NetworkContext.ListContexts()
	if err != nil {
		return s.networkContextError(c, err)
	}
	return c.JSON(http.StatusOK, contexts)
}

func (s *Server) networkContextError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, networkcontext.ErrDeviceNotFound):
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	case errors.Is(err, networkcontext.ErrNoWAN),
		errors.Is(err, networkcontext.ErrInvalidInterface),
		errors.Is(err, networkcontext.ErrDuplicateInterface),
		errors.Is(err, networkcontext.ErrMultipleDefaultRoutes),
		errors.Is(err, networkcontext.ErrInvalidWANRole),
		errors.Is(err, networkcontext.ErrInvalidIPAddress),
		errors.Is(err, networkcontext.ErrInvalidPublicIPv4),
		errors.Is(err, networkcontext.ErrInvalidNetworkName),
		errors.Is(err, networkcontext.ErrInvalidVLAN),
		errors.Is(err, networkcontext.ErrInvalidCIDR),
		errors.Is(err, networkcontext.ErrInvalidGateway):
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	default:
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
}
