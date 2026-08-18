package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/mca-rolando/HermesDDNS/internal/agentauth"
	"github.com/mca-rolando/HermesDDNS/internal/model"
)

const agentAuthContextKey = "hermes.agent.auth"

type agentStageCandidateRequest struct {
	KeyID      string `json:"key_id" validate:"required"`
	SecretHash string `json:"secret_hash" validate:"required"`
}

func (s *Server) issueAgentCredential(c echo.Context) error {
	deviceID, err := parseUintParam(c, "id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid device id"})
	}

	stored, key, err := s.AgentAuth.IssueCredential(deviceID)
	if err != nil {
		return s.agentCredentialAdminError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"credential": stored,
		"agent_key":  key.Plaintext,
		"warning":    "Agent identity key is returned once. Store it securely on the device; Hermes stores only its hash.",
	})
}

func (s *Server) listAgentCredentials(c echo.Context) error {
	deviceID, err := parseUintParam(c, "id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid device id"})
	}
	if err := s.ensureDeviceExists(deviceID); err != nil {
		return s.credentialError(c, err)
	}

	var credentials []model.DeviceIdentityCredential
	if err := s.DB.Where("device_id = ?", deviceID).Order("created_at desc").Find(&credentials).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, credentials)
}

func (s *Server) revokeAgentCredential(c echo.Context) error {
	deviceID, err := parseUintParam(c, "id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid device id"})
	}
	credentialID, err := parseUintParam(c, "credential_id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid agent credential id"})
	}

	if err := s.AgentAuth.RevokeCredential(deviceID, credentialID); err != nil {
		return s.agentCredentialAdminError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "agent credential revoked"})
}

func (s *Server) authenticateAgent(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		token, ok := bearerToken(c.Request().Header.Get("Authorization"))
		if !ok || s.AgentAuth == nil {
			return agentUnauthorized(c)
		}

		auth, err := s.AgentAuth.Authenticate(token, s.callerIP(c))
		if err != nil {
			return agentUnauthorized(c)
		}
		c.Set(agentAuthContextKey, auth)
		return next(c)
	}
}

func (s *Server) agentMe(c echo.Context) error {
	auth, ok := agentContext(c)
	if !ok {
		return agentUnauthorized(c)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"device":     auth.Device,
		"credential": auth.Credential,
	})
}

func (s *Server) agentCurrentRotation(c echo.Context) error {
	auth, ok := agentContext(c)
	if !ok {
		return agentUnauthorized(c)
	}

	rotation, err := s.Credentials.CurrentRotation(auth.Device.ID)
	if err != nil {
		return s.credentialError(c, err)
	}
	if rotation == nil {
		return c.NoContent(http.StatusNoContent)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"rotation":    rotation,
		"next_action": agentRotationNextAction(rotation.Status),
	})
}

func (s *Server) agentStageCandidate(c echo.Context) error {
	auth, ok := agentContext(c)
	if !ok {
		return agentUnauthorized(c)
	}
	rotationID, err := parseUintParam(c, "rotation_id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid rotation id"})
	}

	var req agentStageCandidateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	candidate, err := s.Credentials.StageCandidate(rotationID, auth.Device.ID, req.KeyID, req.SecretHash)
	if err != nil {
		return s.credentialError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"credential":  candidate,
		"next_action": "install_candidate_then_start_validation",
	})
}

func (s *Server) agentStartValidation(c echo.Context) error {
	auth, ok := agentContext(c)
	if !ok {
		return agentUnauthorized(c)
	}
	rotationID, err := parseUintParam(c, "rotation_id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid rotation id"})
	}

	rotation, err := s.Credentials.StartValidation(rotationID, auth.Device.ID)
	if err != nil {
		return s.credentialError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"rotation":    rotation,
		"next_action": "perform_ddns_update_with_candidate",
	})
}

func (s *Server) agentCredentialAdminError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, agentauth.ErrDeviceNotFound),
		errors.Is(err, agentauth.ErrCredentialNotFound):
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	case errors.Is(err, agentauth.ErrActiveCredentialExists):
		return c.JSON(http.StatusConflict, map[string]string{"error": err.Error()})
	default:
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
}

func agentContext(c echo.Context) (agentauth.Context, bool) {
	value := c.Get(agentAuthContextKey)
	auth, ok := value.(agentauth.Context)
	return auth, ok
}

func bearerToken(header string) (string, bool) {
	parts := strings.Fields(strings.TrimSpace(header))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func agentUnauthorized(c echo.Context) error {
	c.Response().Header().Set("WWW-Authenticate", `Bearer realm="HermesDDNS Agent"`)
	return c.JSON(http.StatusUnauthorized, map[string]string{"error": "badauth"})
}

func agentRotationNextAction(status string) string {
	switch status {
	case model.RotationStatusRequested:
		return "generate_and_stage_candidate"
	case model.RotationStatusStaged:
		return "install_candidate_then_start_validation"
	case model.RotationStatusValidating:
		return "perform_ddns_update_with_candidate"
	case model.RotationStatusGrace:
		return "keep_candidate_active_until_grace_completes"
	default:
		return "none"
	}
}
