package api

import (
	"errors"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/mca-rolando/HermesDDNS/internal/agentauth"
	"github.com/mca-rolando/HermesDDNS/internal/enrollment"
)

type createAgentEnrollmentRequest struct {
	TTLMinutes int `json:"ttl_minutes" validate:"omitempty,min=1,max=1440"`
}

type confirmAgentEnrollmentRequest struct {
	AgentVersion string `json:"agent_version" validate:"omitempty,max=128"`
}

type enrollmentDDNSConfiguration struct {
	Username       string   `json:"username"`
	UpdatePath     string   `json:"update_path"`
	AllowedDomains []string `json:"allowed_domains"`
}

func (s *Server) createAgentEnrollment(c echo.Context) error {
	deviceID, err := parseUintParam(c, "id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid device id"})
	}

	var req createAgentEnrollmentRequest
	if err := c.Bind(&req); err != nil && !errors.Is(err, io.EOF) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if req.TTLMinutes == 0 {
		req.TTLMinutes = enrollment.DefaultTTLMinutes
	}

	stored, key, err := s.Enrollments.Create(deviceID, req.TTLMinutes)
	if err != nil {
		return s.enrollmentAdminError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"enrollment":       stored,
		"enrollment_token": key.Plaintext,
		"warning":          "Enrollment token is short-lived, single-use, and returned once. Hermes stores only its hash.",
	})
}

func (s *Server) listAgentEnrollments(c echo.Context) error {
	deviceID, err := parseUintParam(c, "id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid device id"})
	}

	enrollments, err := s.Enrollments.List(deviceID)
	if err != nil {
		return s.enrollmentAdminError(c, err)
	}
	return c.JSON(http.StatusOK, enrollments)
}

func (s *Server) getAgentEnrollment(c echo.Context) error {
	deviceID, err := parseUintParam(c, "id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid device id"})
	}
	enrollmentID, err := parseUintParam(c, "enrollment_id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid enrollment id"})
	}

	stored, err := s.Enrollments.Get(deviceID, enrollmentID)
	if err != nil {
		return s.enrollmentAdminError(c, err)
	}
	return c.JSON(http.StatusOK, stored)
}

func (s *Server) revokeAgentEnrollment(c echo.Context) error {
	deviceID, err := parseUintParam(c, "id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid device id"})
	}
	enrollmentID, err := parseUintParam(c, "enrollment_id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid enrollment id"})
	}

	stored, err := s.Enrollments.Revoke(deviceID, enrollmentID)
	if err != nil {
		return s.enrollmentAdminError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"enrollment": stored,
		"message":    "agent enrollment revoked",
	})
}

func (s *Server) exchangeAgentEnrollment(c echo.Context) error {
	token, ok := bearerToken(c.Request().Header.Get("Authorization"))
	if !ok || s.Enrollments == nil {
		return enrollmentUnauthorized(c)
	}

	result, err := s.Enrollments.Exchange(token, s.callerIP(c))
	if err != nil {
		if errors.Is(err, enrollment.ErrBadAuth) {
			return enrollmentUnauthorized(c)
		}
		return s.enrollmentAdminError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"enrollment":       result.Enrollment,
		"device":           result.Device,
		"agent_credential": result.Credential,
		"agent_key":        result.AgentKey.Plaintext,
		"ddns_configuration": enrollmentDDNSConfiguration{
			Username:       result.Device.Name,
			UpdatePath:     "/nic/update",
			AllowedDomains: s.Config.AllowedDomains,
		},
		"confirmation_path": s.Config.APIPrefix + "/agent/enrollment/confirm",
		"warning":           "Agent identity key is returned once. Persist it securely before confirming enrollment. No DDNS plaintext secret is returned by this endpoint.",
	})
}

func (s *Server) confirmAgentEnrollment(c echo.Context) error {
	auth, ok := agentContext(c)
	if !ok || s.Enrollments == nil {
		return agentUnauthorized(c)
	}

	var req confirmAgentEnrollmentRequest
	if err := c.Bind(&req); err != nil && !errors.Is(err, io.EOF) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	stored, err := s.Enrollments.Confirm(
		auth.Device.ID,
		auth.Credential.ID,
		s.callerIP(c),
		req.AgentVersion,
	)
	if err != nil {
		return s.enrollmentAdminError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"enrollment":  stored,
		"next_action": "enrollment_complete",
	})
}

func (s *Server) enrollmentAdminError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, enrollment.ErrDeviceNotFound),
		errors.Is(err, enrollment.ErrEnrollmentNotFound),
		errors.Is(err, agentauth.ErrDeviceNotFound):
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	case errors.Is(err, enrollment.ErrInvalidTTL):
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	case errors.Is(err, enrollment.ErrEnrollmentInProgress),
		errors.Is(err, enrollment.ErrInvalidState),
		errors.Is(err, agentauth.ErrActiveCredentialExists),
		errors.Is(err, agentauth.ErrCredentialNotFound):
		return c.JSON(http.StatusConflict, map[string]string{"error": err.Error()})
	default:
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
}

func enrollmentUnauthorized(c echo.Context) error {
	c.Response().Header().Set("WWW-Authenticate", `Bearer realm="HermesDDNS Enrollment"`)
	return c.JSON(http.StatusUnauthorized, map[string]string{"error": "badauth"})
}
