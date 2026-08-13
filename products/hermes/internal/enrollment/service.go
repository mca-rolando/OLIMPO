package enrollment

import (
	"errors"
	"fmt"
	"time"

	"github.com/mca-rolando/HermesDDNS/internal/agentauth"
	"github.com/mca-rolando/HermesDDNS/internal/model"
	"github.com/mca-rolando/HermesDDNS/internal/security"
	"gorm.io/gorm"
)

var (
	ErrBadAuth              = errors.New("bad enrollment authentication")
	ErrDeviceNotFound       = errors.New("device not found")
	ErrEnrollmentNotFound   = errors.New("enrollment not found")
	ErrEnrollmentInProgress = errors.New("agent enrollment already in progress")
	ErrInvalidState         = errors.New("invalid enrollment state")
	ErrInvalidTTL           = errors.New("invalid enrollment token lifetime")
)

const (
	DefaultTTLMinutes = 15
	MinTTLMinutes     = 1
	MaxTTLMinutes     = 1440
)

type ExchangeResult struct {
	Enrollment model.AgentEnrollment
	Device     model.Device
	Credential model.DeviceIdentityCredential
	AgentKey   security.GeneratedKey
}

type Service struct {
	DB        *gorm.DB
	AgentAuth *agentauth.Service
	Now       func() time.Time
}

func (s *Service) Create(deviceID uint, ttlMinutes int) (model.AgentEnrollment, security.GeneratedKey, error) {
	if ttlMinutes < MinTTLMinutes || ttlMinutes > MaxTTLMinutes {
		return model.AgentEnrollment{}, security.GeneratedKey{}, ErrInvalidTTL
	}

	var enrollment model.AgentEnrollment
	var generated security.GeneratedKey

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		now := s.now()
		if err := expirePending(tx, deviceID, now); err != nil {
			return err
		}

		var device model.Device
		if err := tx.Where("id = ? AND status = ?", deviceID, "active").First(&device).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrDeviceNotFound
			}
			return err
		}

		var openCount int64
		if err := tx.Model(&model.AgentEnrollment{}).
			Where("device_id = ? AND status IN ?", deviceID, openStatuses()).
			Count(&openCount).Error; err != nil {
			return err
		}
		if openCount != 0 {
			return ErrEnrollmentInProgress
		}

		var activeCredentials int64
		if err := tx.Model(&model.DeviceIdentityCredential{}).
			Where("device_id = ? AND status = ?", deviceID, model.CredentialStatusActive).
			Count(&activeCredentials).Error; err != nil {
			return err
		}
		if activeCredentials != 0 {
			return agentauth.ErrActiveCredentialExists
		}

		key, err := security.GenerateEnrollmentKey()
		if err != nil {
			return err
		}
		generated = key

		enrollment = model.AgentEnrollment{
			DeviceID:   deviceID,
			TokenID:    key.ID,
			SecretHash: key.Hash,
			Status:     model.EnrollmentStatusPending,
			ExpiresAt:  now.Add(time.Duration(ttlMinutes) * time.Minute),
		}
		if err := tx.Create(&enrollment).Error; err != nil {
			return fmt.Errorf("create enrollment: %w", err)
		}
		return nil
	})
	if err != nil {
		return model.AgentEnrollment{}, security.GeneratedKey{}, err
	}

	return enrollment, generated, nil
}

func (s *Service) Exchange(token, callerIP string) (ExchangeResult, error) {
	tokenID, ok := security.ParseEnrollmentKeyID(token)
	if !ok {
		return ExchangeResult{}, ErrBadAuth
	}

	var result ExchangeResult
	var resultErr error

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		var enrollment model.AgentEnrollment
		if err := tx.Where("token_id = ?", tokenID).First(&enrollment).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				resultErr = ErrBadAuth
				return nil
			}
			return err
		}

		if enrollment.Status != model.EnrollmentStatusPending || !security.VerifyAPIKey(token, enrollment.SecretHash) {
			resultErr = ErrBadAuth
			return nil
		}

		now := s.now()
		if !enrollment.ExpiresAt.After(now) {
			if err := tx.Model(&enrollment).Where("status = ?", model.EnrollmentStatusPending).Updates(map[string]any{
				"status": model.EnrollmentStatusExpired,
			}).Error; err != nil {
				return err
			}
			resultErr = ErrBadAuth
			return nil
		}

		// Atomically claim the one-time token before issuing a permanent
		// credential. If any later step fails, the transaction rolls the claim
		// back to pending. A competing exchange cannot successfully claim it.
		claimed := tx.Model(&model.AgentEnrollment{}).
			Where("id = ? AND status = ?", enrollment.ID, model.EnrollmentStatusPending).
			Updates(map[string]any{
				"status":    model.EnrollmentStatusIssued,
				"issued_at": now,
				"used_ip":   callerIP,
			})
		if claimed.Error != nil {
			return claimed.Error
		}
		if claimed.RowsAffected != 1 {
			return ErrBadAuth
		}

		credential, agentKey, err := s.agentAuth().IssueCredentialTx(tx, enrollment.DeviceID)
		if err != nil {
			return err
		}
		if err := tx.Model(&model.AgentEnrollment{}).Where("id = ?", enrollment.ID).Update("agent_credential_id", credential.ID).Error; err != nil {
			return err
		}

		enrollment.Status = model.EnrollmentStatusIssued
		enrollment.IssuedAt = &now
		enrollment.UsedIP = callerIP
		enrollment.AgentCredentialID = &credential.ID

		var device model.Device
		if err := tx.First(&device, enrollment.DeviceID).Error; err != nil {
			return err
		}

		result = ExchangeResult{
			Enrollment: enrollment,
			Device:     device,
			Credential: credential,
			AgentKey:   agentKey,
		}
		return nil
	})
	if err != nil {
		return ExchangeResult{}, err
	}
	if resultErr != nil {
		return ExchangeResult{}, resultErr
	}

	return result, nil
}

func (s *Service) Confirm(deviceID, agentCredentialID uint, callerIP, agentVersion string) (model.AgentEnrollment, error) {
	var result model.AgentEnrollment

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		var enrollment model.AgentEnrollment
		if err := tx.Where(
			"device_id = ? AND agent_credential_id = ? AND status IN ?",
			deviceID,
			agentCredentialID,
			[]string{model.EnrollmentStatusIssued, model.EnrollmentStatusCompleted},
		).Order("created_at desc").First(&enrollment).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrEnrollmentNotFound
			}
			return err
		}

		now := s.now()
		if enrollment.Status == model.EnrollmentStatusIssued {
			updated := tx.Model(&model.AgentEnrollment{}).
				Where("id = ? AND status = ?", enrollment.ID, model.EnrollmentStatusIssued).
				Updates(map[string]any{
					"status":       model.EnrollmentStatusCompleted,
					"completed_at": now,
				})
			if updated.Error != nil {
				return updated.Error
			}
			if updated.RowsAffected != 1 {
				return ErrInvalidState
			}
			enrollment.Status = model.EnrollmentStatusCompleted
			enrollment.CompletedAt = &now
		}

		deviceUpdates := map[string]any{
			"last_seen_at": now,
			"last_ip":      callerIP,
		}
		if agentVersion != "" {
			deviceUpdates["agent_version"] = agentVersion
		}
		if err := tx.Model(&model.Device{}).Where("id = ?", deviceID).Updates(deviceUpdates).Error; err != nil {
			return err
		}

		result = enrollment
		return nil
	})
	if err != nil {
		return model.AgentEnrollment{}, err
	}

	return result, nil
}

func (s *Service) List(deviceID uint) ([]model.AgentEnrollment, error) {
	if err := s.ensureDevice(deviceID); err != nil {
		return nil, err
	}
	if err := s.reconcileExpired(deviceID); err != nil {
		return nil, err
	}

	var enrollments []model.AgentEnrollment
	if err := s.DB.Where("device_id = ?", deviceID).Order("created_at desc").Find(&enrollments).Error; err != nil {
		return nil, err
	}
	return enrollments, nil
}

func (s *Service) Get(deviceID, enrollmentID uint) (model.AgentEnrollment, error) {
	if err := s.reconcileExpired(deviceID); err != nil {
		return model.AgentEnrollment{}, err
	}

	var enrollment model.AgentEnrollment
	if err := s.DB.Where("id = ? AND device_id = ?", enrollmentID, deviceID).First(&enrollment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.AgentEnrollment{}, ErrEnrollmentNotFound
		}
		return model.AgentEnrollment{}, err
	}
	return enrollment, nil
}

func (s *Service) Revoke(deviceID, enrollmentID uint) (model.AgentEnrollment, error) {
	if err := s.reconcileExpired(deviceID); err != nil {
		return model.AgentEnrollment{}, err
	}

	var result model.AgentEnrollment

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		var enrollment model.AgentEnrollment
		if err := tx.Where("id = ? AND device_id = ?", enrollmentID, deviceID).First(&enrollment).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrEnrollmentNotFound
			}
			return err
		}

		now := s.now()

		switch enrollment.Status {
		case model.EnrollmentStatusRevoked:
			result = enrollment
			return nil
		case model.EnrollmentStatusPending:
			// No permanent identity has been issued yet.
		case model.EnrollmentStatusIssued:
			if enrollment.AgentCredentialID == nil {
				return ErrInvalidState
			}
			if err := s.agentAuth().RevokeCredentialTx(tx, deviceID, *enrollment.AgentCredentialID); err != nil {
				if !errors.Is(err, agentauth.ErrCredentialNotFound) {
					return err
				}
				var credential model.DeviceIdentityCredential
				if lookupErr := tx.Where(
					"id = ? AND device_id = ? AND status = ?",
					*enrollment.AgentCredentialID,
					deviceID,
					model.CredentialStatusRevoked,
				).First(&credential).Error; lookupErr != nil {
					return ErrInvalidState
				}
			}
		default:
			return ErrInvalidState
		}

		updated := tx.Model(&model.AgentEnrollment{}).
			Where("id = ? AND status IN ?", enrollment.ID, []string{model.EnrollmentStatusPending, model.EnrollmentStatusIssued}).
			Updates(map[string]any{
				"status":     model.EnrollmentStatusRevoked,
				"revoked_at": now,
			})
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected != 1 {
			return ErrInvalidState
		}

		enrollment.Status = model.EnrollmentStatusRevoked
		enrollment.RevokedAt = &now
		result = enrollment
		return nil
	})
	if err != nil {
		return model.AgentEnrollment{}, err
	}

	return result, nil
}

func (s *Service) reconcileExpired(deviceID uint) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		return expirePending(tx, deviceID, s.now())
	})
}

func (s *Service) ensureDevice(deviceID uint) error {
	var count int64
	if err := s.DB.Model(&model.Device{}).Where("id = ?", deviceID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrDeviceNotFound
	}
	return nil
}

func (s *Service) agentAuth() *agentauth.Service {
	if s.AgentAuth != nil {
		return s.AgentAuth
	}
	return &agentauth.Service{DB: s.DB, Now: s.Now}
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func expirePending(tx *gorm.DB, deviceID uint, now time.Time) error {
	query := tx.Model(&model.AgentEnrollment{}).
		Where("status = ? AND expires_at <= ?", model.EnrollmentStatusPending, now)
	if deviceID != 0 {
		query = query.Where("device_id = ?", deviceID)
	}
	return query.Update("status", model.EnrollmentStatusExpired).Error
}

func openStatuses() []string {
	return []string{model.EnrollmentStatusPending, model.EnrollmentStatusIssued}
}
