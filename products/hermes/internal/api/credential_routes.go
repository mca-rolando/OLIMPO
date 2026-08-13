package api

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/mca-rolando/HermesDDNS/internal/credential"
	"github.com/mca-rolando/HermesDDNS/internal/model"
	"gorm.io/gorm"
)

type credentialRotationRequest struct {
	GraceMinutes int `json:"grace_minutes" validate:"omitempty,min=1,max=1440"`
}

type credentialRotationRollbackRequest struct {
	Reason string `json:"reason"`
}

func (s *Server) requestCredentialRotation(c echo.Context) error {
	deviceID, err := parseUintParam(c, "id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid device id"})
	}

	var req credentialRotationRequest
	if err := c.Bind(&req); err != nil && !errors.Is(err, io.EOF) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if req.GraceMinutes == 0 {
		req.GraceMinutes = 30
	}

	rotation, err := s.Credentials.RequestRotation(deviceID, req.GraceMinutes)
	if err != nil {
		return s.credentialError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"rotation":    rotation,
		"next_action": "agent_stage_candidate",
		"warning":     "Rotation requested. The current DDNS credential remains active until the replacement credential is validated by the device.",
	})
}

func (s *Server) listCredentialRotations(c echo.Context) error {
	deviceID, err := parseUintParam(c, "id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid device id"})
	}

	if err := s.ensureDeviceExists(deviceID); err != nil {
		return s.credentialError(c, err)
	}

	var rotations []model.DDNSCredentialRotation
	if err := s.DB.Where("device_id = ?", deviceID).Order("created_at desc").Find(&rotations).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, rotations)
}

func (s *Server) getCredentialRotation(c echo.Context) error {
	deviceID, err := parseUintParam(c, "id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid device id"})
	}
	rotationID, err := parseUintParam(c, "rotation_id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid rotation id"})
	}

	var rotation model.DDNSCredentialRotation
	if err := s.DB.Where("id = ? AND device_id = ?", rotationID, deviceID).First(&rotation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": credential.ErrRotationNotFound.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, rotation)
}

func (s *Server) rollbackCredentialRotation(c echo.Context) error {
	deviceID, err := parseUintParam(c, "id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid device id"})
	}
	rotationID, err := parseUintParam(c, "rotation_id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid rotation id"})
	}

	var req credentialRotationRollbackRequest
	if err := c.Bind(&req); err != nil && !errors.Is(err, io.EOF) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if req.Reason == "" {
		req.Reason = "administrator requested rollback"
	}

	if err := s.Credentials.Rollback(rotationID, deviceID, req.Reason); err != nil {
		return s.credentialError(c, err)
	}

	var rotation model.DDNSCredentialRotation
	if err := s.DB.First(&rotation, rotationID).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"rotation": rotation,
		"message":  "credential rotation rolled back",
	})
}

func (s *Server) reconcileCredentialRotations(c echo.Context) error {
	completed, err := s.Credentials.ReconcileExpiredGrace()
	if err != nil {
		return s.credentialError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"completed": completed,
	})
}

func (s *Server) ensureDeviceExists(deviceID uint) error {
	var count int64
	if err := s.DB.Model(&model.Device{}).Where("id = ?", deviceID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return credential.ErrDeviceNotFound
	}
	return nil
}

func (s *Server) credentialError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, credential.ErrDeviceNotFound),
		errors.Is(err, credential.ErrRotationNotFound):
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	case errors.Is(err, credential.ErrInvalidGracePeriod),
		errors.Is(err, credential.ErrInvalidCandidateKeyID),
		errors.Is(err, credential.ErrInvalidCandidateHash):
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	case errors.Is(err, credential.ErrNoActiveCredential),
		errors.Is(err, credential.ErrMultipleActive),
		errors.Is(err, credential.ErrRotationInProgress),
		errors.Is(err, credential.ErrInvalidRotationState):
		return c.JSON(http.StatusConflict, map[string]string{"error": err.Error()})
	default:
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
}

func parseUintParam(c echo.Context, name string) (uint, error) {
	value, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(value), nil
}
