package credential

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mca-rolando/HermesDDNS/internal/model"
	"gorm.io/gorm"
)

var (
	ErrDeviceNotFound        = errors.New("device not found")
	ErrNoActiveCredential    = errors.New("no active DDNS credential")
	ErrMultipleActive        = errors.New("multiple active DDNS credentials")
	ErrRotationInProgress    = errors.New("credential rotation already in progress")
	ErrRotationNotFound      = errors.New("credential rotation not found")
	ErrInvalidRotationState  = errors.New("invalid credential rotation state")
	ErrInvalidGracePeriod    = errors.New("invalid grace period")
	ErrInvalidCandidateKeyID = errors.New("invalid candidate key id")
	ErrInvalidCandidateHash  = errors.New("invalid candidate secret hash")
)

const (
	MinGraceMinutes = 1
	MaxGraceMinutes = 1440
)

type Service struct {
	DB  *gorm.DB
	Now func() time.Time
}

func (s *Service) RequestRotation(deviceID uint, graceMinutes int) (model.DDNSCredentialRotation, error) {
	if graceMinutes < MinGraceMinutes || graceMinutes > MaxGraceMinutes {
		return model.DDNSCredentialRotation{}, ErrInvalidGracePeriod
	}

	var rotation model.DDNSCredentialRotation
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		var device model.Device
		if err := tx.Where("id = ? AND status = ?", deviceID, "active").First(&device).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrDeviceNotFound
			}
			return err
		}

		var openCount int64
		if err := tx.Model(&model.DDNSCredentialRotation{}).
			Where("device_id = ? AND status IN ?", deviceID, openRotationStatuses()).
			Count(&openCount).Error; err != nil {
			return err
		}
		if openCount != 0 {
			return ErrRotationInProgress
		}

		var active []model.DDNSCredential
		if err := tx.Where("device_id = ? AND status = ?", deviceID, model.CredentialStatusActive).
			Order("id desc").Limit(2).Find(&active).Error; err != nil {
			return err
		}
		switch len(active) {
		case 0:
			return ErrNoActiveCredential
		case 1:
		default:
			return ErrMultipleActive
		}

		now := s.now()
		rotation = model.DDNSCredentialRotation{
			DeviceID:             deviceID,
			PreviousCredentialID: active[0].ID,
			Status:               model.RotationStatusRequested,
			GraceMinutes:         graceMinutes,
			RequestedAt:          now,
		}
		return tx.Create(&rotation).Error
	})
	if err != nil {
		return model.DDNSCredentialRotation{}, err
	}
	return rotation, nil
}

func (s *Service) StageCandidate(rotationID, deviceID uint, keyID, secretHash string) (model.DDNSCredential, error) {
	keyID = strings.ToLower(strings.TrimSpace(keyID))
	secretHash = strings.ToLower(strings.TrimSpace(secretHash))

	if !validHexBytes(keyID, 8) {
		return model.DDNSCredential{}, ErrInvalidCandidateKeyID
	}
	if !validHexBytes(secretHash, 32) {
		return model.DDNSCredential{}, ErrInvalidCandidateHash
	}

	var candidate model.DDNSCredential
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		rotation, err := loadRotation(tx, rotationID, deviceID)
		if err != nil {
			return err
		}
		if rotation.Status != model.RotationStatusRequested || rotation.NewCredentialID != nil {
			return ErrInvalidRotationState
		}

		var previous model.DDNSCredential
		if err := tx.Where(
			"id = ? AND device_id = ? AND status = ?",
			rotation.PreviousCredentialID,
			deviceID,
			model.CredentialStatusActive,
		).First(&previous).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvalidRotationState
			}
			return err
		}

		now := s.now()
		previousID := previous.ID
		candidate = model.DDNSCredential{
			DeviceID:             deviceID,
			KeyID:                keyID,
			SecretHash:           secretHash,
			Status:               model.CredentialStatusPending,
			ReplacesCredentialID: &previousID,
		}
		if err := tx.Create(&candidate).Error; err != nil {
			return fmt.Errorf("create candidate credential: %w", err)
		}

		return tx.Model(&rotation).Updates(map[string]any{
			"new_credential_id": candidate.ID,
			"status":            model.RotationStatusStaged,
			"staged_at":         now,
		}).Error
	})
	if err != nil {
		return model.DDNSCredential{}, err
	}
	return candidate, nil
}

func (s *Service) StartValidation(rotationID, deviceID uint) (model.DDNSCredentialRotation, error) {
	var result model.DDNSCredentialRotation
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		rotation, err := loadRotation(tx, rotationID, deviceID)
		if err != nil {
			return err
		}
		if rotation.Status != model.RotationStatusStaged || rotation.NewCredentialID == nil {
			return ErrInvalidRotationState
		}

		now := s.now()
		updates := tx.Model(&model.DDNSCredential{}).
			Where(
				"id = ? AND device_id = ? AND status = ?",
				*rotation.NewCredentialID,
				deviceID,
				model.CredentialStatusPending,
			).
			Updates(map[string]any{
				"status":       model.CredentialStatusActive,
				"activated_at": now,
			})
		if updates.Error != nil {
			return updates.Error
		}
		if updates.RowsAffected != 1 {
			return ErrInvalidRotationState
		}

		if err := tx.Model(&rotation).Updates(map[string]any{
			"status":                model.RotationStatusValidating,
			"validation_started_at": now,
		}).Error; err != nil {
			return err
		}

		return tx.First(&result, rotation.ID).Error
	})
	if err != nil {
		return model.DDNSCredentialRotation{}, err
	}
	return result, nil
}

// ConfirmCredentialUse advances a validating rotation only after the replacement
// credential has successfully authenticated to the DDNS endpoint. Ordinary
// credentials that are not part of a validating rotation are a no-op.
func (s *Service) ConfirmCredentialUse(credentialID uint) (bool, error) {
	advanced := false
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		var rotation model.DDNSCredentialRotation
		if err := tx.Where(
			"new_credential_id = ? AND status = ?",
			credentialID,
			model.RotationStatusValidating,
		).First(&rotation).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}

		var replacement model.DDNSCredential
		if err := tx.Where(
			"id = ? AND device_id = ? AND status = ?",
			credentialID,
			rotation.DeviceID,
			model.CredentialStatusActive,
		).First(&replacement).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvalidRotationState
			}
			return err
		}

		now := s.now()
		graceUntil := now.Add(time.Duration(rotation.GraceMinutes) * time.Minute)

		if err := tx.Model(&replacement).Updates(map[string]any{
			"confirmed_at": now,
		}).Error; err != nil {
			return err
		}

		previous := tx.Model(&model.DDNSCredential{}).
			Where(
				"id = ? AND device_id = ? AND status = ?",
				rotation.PreviousCredentialID,
				rotation.DeviceID,
				model.CredentialStatusActive,
			).
			Updates(map[string]any{
				"status":      model.CredentialStatusGrace,
				"grace_until": graceUntil,
			})
		if previous.Error != nil {
			return previous.Error
		}
		if previous.RowsAffected != 1 {
			return ErrInvalidRotationState
		}

		if err := tx.Model(&rotation).Updates(map[string]any{
			"status":       model.RotationStatusGrace,
			"confirmed_at": now,
			"grace_until":  graceUntil,
		}).Error; err != nil {
			return err
		}

		advanced = true
		return nil
	})
	return advanced, err
}

func (s *Service) ReconcileExpiredGrace() (int, error) {
	now := s.now()
	var rotations []model.DDNSCredentialRotation
	if err := s.DB.Where(
		"status = ? AND grace_until IS NOT NULL AND grace_until <= ?",
		model.RotationStatusGrace,
		now,
	).Find(&rotations).Error; err != nil {
		return 0, err
	}

	completed := 0
	for _, item := range rotations {
		err := s.DB.Transaction(func(tx *gorm.DB) error {
			var rotation model.DDNSCredentialRotation
			if err := tx.Where("id = ? AND status = ?", item.ID, model.RotationStatusGrace).
				First(&rotation).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil
				}
				return err
			}
			if rotation.GraceUntil == nil || rotation.GraceUntil.After(now) {
				return nil
			}

			old := tx.Model(&model.DDNSCredential{}).
				Where(
					"id = ? AND device_id = ? AND status = ?",
					rotation.PreviousCredentialID,
					rotation.DeviceID,
					model.CredentialStatusGrace,
				).
				Updates(map[string]any{
					"status":     model.CredentialStatusRevoked,
					"revoked_at": now,
				})
			if old.Error != nil {
				return old.Error
			}
			if old.RowsAffected != 1 {
				return ErrInvalidRotationState
			}

			if err := tx.Model(&rotation).Updates(map[string]any{
				"status":       model.RotationStatusCompleted,
				"completed_at": now,
			}).Error; err != nil {
				return err
			}

			return nil
		})
		if err != nil {
			return completed, err
		}
		completed++
	}

	return completed, nil
}

func (s *Service) Rollback(rotationID, deviceID uint, reason string) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		rotation, err := loadRotation(tx, rotationID, deviceID)
		if err != nil {
			return err
		}

		switch rotation.Status {
		case model.RotationStatusRequested:
			// No replacement credential exists yet.
		case model.RotationStatusStaged:
			if rotation.NewCredentialID == nil {
				return ErrInvalidRotationState
			}
			if err := revokeReplacement(tx, *rotation.NewCredentialID, deviceID, model.CredentialStatusPending, s.now()); err != nil {
				return err
			}
		case model.RotationStatusValidating:
			if rotation.NewCredentialID == nil {
				return ErrInvalidRotationState
			}
			if err := revokeReplacement(tx, *rotation.NewCredentialID, deviceID, model.CredentialStatusActive, s.now()); err != nil {
				return err
			}
		case model.RotationStatusGrace:
			if rotation.NewCredentialID == nil {
				return ErrInvalidRotationState
			}
			now := s.now()
			if rotation.GraceUntil == nil || !rotation.GraceUntil.After(now) {
				return ErrInvalidRotationState
			}
			old := tx.Model(&model.DDNSCredential{}).
				Where(
					"id = ? AND device_id = ? AND status = ?",
					rotation.PreviousCredentialID,
					deviceID,
					model.CredentialStatusGrace,
				).
				Updates(map[string]any{
					"status":      model.CredentialStatusActive,
					"grace_until": nil,
				})
			if old.Error != nil {
				return old.Error
			}
			if old.RowsAffected != 1 {
				return ErrInvalidRotationState
			}
			if err := revokeReplacement(tx, *rotation.NewCredentialID, deviceID, model.CredentialStatusActive, now); err != nil {
				return err
			}
		default:
			return ErrInvalidRotationState
		}

		now := s.now()
		return tx.Model(&rotation).Updates(map[string]any{
			"status":         model.RotationStatusRolledBack,
			"rolled_back_at": now,
			"last_error":     reason,
		}).Error
	})
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func openRotationStatuses() []string {
	return []string{
		model.RotationStatusRequested,
		model.RotationStatusStaged,
		model.RotationStatusValidating,
		model.RotationStatusGrace,
	}
}

func loadRotation(tx *gorm.DB, rotationID, deviceID uint) (model.DDNSCredentialRotation, error) {
	var rotation model.DDNSCredentialRotation
	if err := tx.Where("id = ? AND device_id = ?", rotationID, deviceID).First(&rotation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.DDNSCredentialRotation{}, ErrRotationNotFound
		}
		return model.DDNSCredentialRotation{}, err
	}
	return rotation, nil
}

func revokeReplacement(tx *gorm.DB, credentialID, deviceID uint, expectedStatus string, now time.Time) error {
	updated := tx.Model(&model.DDNSCredential{}).
		Where("id = ? AND device_id = ? AND status = ?", credentialID, deviceID, expectedStatus).
		Updates(map[string]any{
			"status":     model.CredentialStatusRevoked,
			"revoked_at": now,
		})
	if updated.Error != nil {
		return updated.Error
	}
	if updated.RowsAffected != 1 {
		return ErrInvalidRotationState
	}
	return nil
}

func validHexBytes(value string, byteCount int) bool {
	if len(value) != byteCount*2 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == byteCount
}
