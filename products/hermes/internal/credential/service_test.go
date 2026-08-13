package credential

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/mca-rolando/HermesDDNS/internal/database"
	"github.com/mca-rolando/HermesDDNS/internal/model"
	"github.com/mca-rolando/HermesDDNS/internal/security"
	"gorm.io/gorm"
)

func TestRotationLifecycle(t *testing.T) {
	clock := time.Date(2026, 8, 13, 15, 0, 0, 0, time.UTC)
	svc, db, device, old := newTestService(t, &clock)

	rotation, err := svc.RequestRotation(device.ID, 30)
	if err != nil {
		t.Fatal(err)
	}
	if rotation.Status != model.RotationStatusRequested {
		t.Fatalf("unexpected requested status: %s", rotation.Status)
	}
	if rotation.PreviousCredentialID != old.ID {
		t.Fatalf("unexpected previous credential: got %d want %d", rotation.PreviousCredentialID, old.ID)
	}

	assertCredentialStatus(t, db, old.ID, model.CredentialStatusActive)

	key, err := security.GenerateDDNSKey()
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := svc.StageCandidate(rotation.ID, device.ID, key.ID, key.Hash)
	if err != nil {
		t.Fatal(err)
	}
	if candidate.Status != model.CredentialStatusPending {
		t.Fatalf("candidate should be pending, got %s", candidate.Status)
	}
	if candidate.ReplacesCredentialID == nil || *candidate.ReplacesCredentialID != old.ID {
		t.Fatalf("candidate does not reference replaced credential")
	}

	clock = clock.Add(time.Minute)
	rotation, err = svc.StartValidation(rotation.ID, device.ID)
	if err != nil {
		t.Fatal(err)
	}
	if rotation.Status != model.RotationStatusValidating {
		t.Fatalf("unexpected validating status: %s", rotation.Status)
	}

	// Validation deliberately keeps both credentials active. The old key must
	// not enter grace until the replacement has actually authenticated.
	assertCredentialStatus(t, db, old.ID, model.CredentialStatusActive)
	assertCredentialStatus(t, db, candidate.ID, model.CredentialStatusActive)

	clock = clock.Add(time.Minute)
	advanced, err := svc.ConfirmCredentialUse(candidate.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !advanced {
		t.Fatal("replacement credential use should advance the rotation")
	}

	var storedRotation model.DDNSCredentialRotation
	if err := db.First(&storedRotation, rotation.ID).Error; err != nil {
		t.Fatal(err)
	}
	if storedRotation.Status != model.RotationStatusGrace {
		t.Fatalf("unexpected grace status: %s", storedRotation.Status)
	}
	if storedRotation.ConfirmedAt == nil || storedRotation.GraceUntil == nil {
		t.Fatal("confirmed rotation must record confirmation and grace deadline")
	}

	expectedGrace := clock.Add(30 * time.Minute)
	if !storedRotation.GraceUntil.Equal(expectedGrace) {
		t.Fatalf("unexpected grace deadline: got %v want %v", storedRotation.GraceUntil, expectedGrace)
	}

	assertCredentialStatus(t, db, old.ID, model.CredentialStatusGrace)
	assertCredentialStatus(t, db, candidate.ID, model.CredentialStatusActive)

	var confirmed model.DDNSCredential
	if err := db.First(&confirmed, candidate.ID).Error; err != nil {
		t.Fatal(err)
	}
	if confirmed.ConfirmedAt == nil || !confirmed.ConfirmedAt.Equal(clock) {
		t.Fatalf("replacement credential was not confirmed at expected time")
	}

	clock = expectedGrace.Add(time.Second)
	completed, err := svc.ReconcileExpiredGrace()
	if err != nil {
		t.Fatal(err)
	}
	if completed != 1 {
		t.Fatalf("expected one completed rotation, got %d", completed)
	}

	assertCredentialStatus(t, db, old.ID, model.CredentialStatusRevoked)
	assertCredentialStatus(t, db, candidate.ID, model.CredentialStatusActive)

	if err := db.First(&storedRotation, rotation.ID).Error; err != nil {
		t.Fatal(err)
	}
	if storedRotation.Status != model.RotationStatusCompleted || storedRotation.CompletedAt == nil {
		t.Fatalf("rotation should be completed: %#v", storedRotation)
	}
}

func TestSecondOpenRotationIsRejected(t *testing.T) {
	clock := time.Date(2026, 8, 13, 15, 0, 0, 0, time.UTC)
	svc, _, device, _ := newTestService(t, &clock)

	if _, err := svc.RequestRotation(device.ID, 30); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RequestRotation(device.ID, 30); !errors.Is(err, ErrRotationInProgress) {
		t.Fatalf("expected ErrRotationInProgress, got %v", err)
	}
}

func TestInvalidGracePeriodsAreRejected(t *testing.T) {
	clock := time.Date(2026, 8, 13, 15, 0, 0, 0, time.UTC)
	svc, _, device, _ := newTestService(t, &clock)

	for _, minutes := range []int{0, -1, MaxGraceMinutes + 1} {
		if _, err := svc.RequestRotation(device.ID, minutes); !errors.Is(err, ErrInvalidGracePeriod) {
			t.Fatalf("grace %d: expected ErrInvalidGracePeriod, got %v", minutes, err)
		}
	}
}

func TestStageCandidateValidatesKeyMaterial(t *testing.T) {
	clock := time.Date(2026, 8, 13, 15, 0, 0, 0, time.UTC)
	svc, _, device, _ := newTestService(t, &clock)
	rotation, err := svc.RequestRotation(device.ID, 30)
	if err != nil {
		t.Fatal(err)
	}

	key, err := security.GenerateDDNSKey()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.StageCandidate(rotation.ID, device.ID, "not-hex", key.Hash); !errors.Is(err, ErrInvalidCandidateKeyID) {
		t.Fatalf("expected ErrInvalidCandidateKeyID, got %v", err)
	}
	if _, err := svc.StageCandidate(rotation.ID, device.ID, key.ID, "not-a-sha256-hash"); !errors.Is(err, ErrInvalidCandidateHash) {
		t.Fatalf("expected ErrInvalidCandidateHash, got %v", err)
	}
}

func TestOrdinaryCredentialUseDoesNotAdvanceRotation(t *testing.T) {
	clock := time.Date(2026, 8, 13, 15, 0, 0, 0, time.UTC)
	svc, _, _, old := newTestService(t, &clock)

	advanced, err := svc.ConfirmCredentialUse(old.ID)
	if err != nil {
		t.Fatal(err)
	}
	if advanced {
		t.Fatal("ordinary active credential must not advance a rotation")
	}
}

func TestRollbackDuringValidationKeepsOldCredentialActive(t *testing.T) {
	clock := time.Date(2026, 8, 13, 15, 0, 0, 0, time.UTC)
	svc, db, device, old := newTestService(t, &clock)

	rotation, err := svc.RequestRotation(device.ID, 30)
	if err != nil {
		t.Fatal(err)
	}
	key, err := security.GenerateDDNSKey()
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := svc.StageCandidate(rotation.ID, device.ID, key.ID, key.Hash)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.StartValidation(rotation.ID, device.ID); err != nil {
		t.Fatal(err)
	}

	clock = clock.Add(5 * time.Minute)
	if err := svc.Rollback(rotation.ID, device.ID, "agent failed to install candidate"); err != nil {
		t.Fatal(err)
	}

	assertCredentialStatus(t, db, old.ID, model.CredentialStatusActive)
	assertCredentialStatus(t, db, candidate.ID, model.CredentialStatusRevoked)

	var stored model.DDNSCredentialRotation
	if err := db.First(&stored, rotation.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != model.RotationStatusRolledBack || stored.RolledBackAt == nil {
		t.Fatalf("rotation should be rolled back: %#v", stored)
	}
	if stored.LastError != "agent failed to install candidate" {
		t.Fatalf("unexpected rollback reason: %q", stored.LastError)
	}
}

func TestRollbackDuringGraceRestoresPreviousCredential(t *testing.T) {
	clock := time.Date(2026, 8, 13, 15, 0, 0, 0, time.UTC)
	svc, db, device, old := newTestService(t, &clock)

	rotation, err := svc.RequestRotation(device.ID, 30)
	if err != nil {
		t.Fatal(err)
	}
	key, err := security.GenerateDDNSKey()
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := svc.StageCandidate(rotation.ID, device.ID, key.ID, key.Hash)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.StartValidation(rotation.ID, device.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ConfirmCredentialUse(candidate.ID); err != nil {
		t.Fatal(err)
	}

	clock = clock.Add(5 * time.Minute)
	if err := svc.Rollback(rotation.ID, device.ID, "operator rollback"); err != nil {
		t.Fatal(err)
	}

	assertCredentialStatus(t, db, old.ID, model.CredentialStatusActive)
	assertCredentialStatus(t, db, candidate.ID, model.CredentialStatusRevoked)

	var restored model.DDNSCredential
	if err := db.First(&restored, old.ID).Error; err != nil {
		t.Fatal(err)
	}
	if restored.GraceUntil != nil {
		t.Fatalf("restored credential should not retain grace deadline: %v", restored.GraceUntil)
	}
}

func newTestService(t *testing.T, clock *time.Time) (*Service, *gorm.DB, model.Device, model.DDNSCredential) {
	t.Helper()

	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}

	device := model.Device{Name: "ROTATION-TEST", Status: "active"}
	if err := db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}

	key, err := security.GenerateDDNSKey()
	if err != nil {
		t.Fatal(err)
	}
	activatedAt := clock.UTC()
	old := model.DDNSCredential{
		DeviceID:    device.ID,
		KeyID:       key.ID,
		SecretHash:  key.Hash,
		Status:      model.CredentialStatusActive,
		ActivatedAt: &activatedAt,
	}
	if err := db.Create(&old).Error; err != nil {
		t.Fatal(err)
	}

	svc := &Service{
		DB: db,
		Now: func() time.Time {
			return clock.UTC()
		},
	}
	return svc, db, device, old
}

func assertCredentialStatus(t *testing.T, db *gorm.DB, id uint, want string) {
	t.Helper()

	var cred model.DDNSCredential
	if err := db.First(&cred, id).Error; err != nil {
		t.Fatal(err)
	}
	if cred.Status != want {
		t.Fatalf("credential %d status: got %s want %s", id, cred.Status, want)
	}
}

func TestCurrentRotationReturnsOpenRotationOrNil(t *testing.T) {
	clock := time.Date(2026, 8, 13, 15, 0, 0, 0, time.UTC)
	svc, _, device, _ := newTestService(t, &clock)

	current, err := svc.CurrentRotation(device.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current != nil {
		t.Fatalf("expected no current rotation, got %#v", current)
	}

	rotation, err := svc.RequestRotation(device.ID, 30)
	if err != nil {
		t.Fatal(err)
	}
	current, err = svc.CurrentRotation(device.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current == nil || current.ID != rotation.ID {
		t.Fatalf("unexpected current rotation: %#v", current)
	}
}

func TestRunReconcilerCompletesExpiredGraceAutomatically(t *testing.T) {
	clock := time.Date(2026, 8, 13, 15, 0, 0, 0, time.UTC)
	svc, db, device, old := newTestService(t, &clock)

	rotation, err := svc.RequestRotation(device.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	key, err := security.GenerateDDNSKey()
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := svc.StageCandidate(rotation.ID, device.ID, key.ID, key.Hash)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.StartValidation(rotation.ID, device.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ConfirmCredentialUse(candidate.ID); err != nil {
		t.Fatal(err)
	}

	clock = clock.Add(2 * time.Minute)

	type reconcileResult struct {
		completed int
		err       error
	}
	results := make(chan reconcileResult, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go svc.RunReconciler(ctx, 10*time.Millisecond, func(completed int, err error) {
		select {
		case results <- reconcileResult{completed: completed, err: err}:
		default:
		}
	})

	select {
	case result := <-results:
		if result.err != nil {
			t.Fatal(result.err)
		}
		if result.completed != 1 {
			t.Fatalf("expected one automatically completed rotation, got %d", result.completed)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for automatic grace reconciliation")
	}
	cancel()

	assertCredentialStatus(t, db, old.ID, model.CredentialStatusRevoked)
	assertCredentialStatus(t, db, candidate.ID, model.CredentialStatusActive)

	var stored model.DDNSCredentialRotation
	if err := db.First(&stored, rotation.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != model.RotationStatusCompleted || stored.CompletedAt == nil {
		t.Fatalf("rotation should be automatically completed: %#v", stored)
	}
}
