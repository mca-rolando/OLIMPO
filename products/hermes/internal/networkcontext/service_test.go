package networkcontext

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/mca-rolando/HermesDDNS/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newNetworkContextService(t *testing.T) (*gorm.DB, *Service, model.Device) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "network.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Device{}, &model.NetworkIdentitySnapshot{}, &model.NetworkWAN{}, &model.NetworkSegment{}); err != nil {
		t.Fatal(err)
	}
	device := model.Device{Name: "COR-P-NETWORK", Status: "active", Type: "UDM-SE"}
	if err := db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 13, 22, 0, 0, 0, time.UTC)
	return db, &Service{DB: db, Now: func() time.Time { return now }}, device
}

func TestReportClassifiesDirectDoubleNATCGNATAndMismatch(t *testing.T) {
	_, service, device := newNetworkContextService(t)
	vlan20 := 20
	context, err := service.Report(device.ID, "73.44.18.91", ReportInput{
		WANs: []WANInput{
			{InterfaceName: "eth8", Role: WANRolePrimary, DefaultRoute: true, IPv4: "73.44.18.91", GatewayIPv4: "73.44.18.89"},
			{InterfaceName: "eth9", Role: WANRoleSecondary, IPv4: "192.168.1.20", GatewayIPv4: "192.168.1.1", PublicIPv4: "8.8.8.8"},
			{InterfaceName: "wwan0", Role: WANRoleOther, IPv4: "100.72.35.18", PublicIPv4: "1.1.1.1"},
			{InterfaceName: "eth10", Role: WANRoleOther, IPv4: "9.9.9.9", PublicIPv4: "8.8.4.4"},
		},
		Networks: []SegmentInput{{Name: "POS", VLANID: &vlan20, IPv4CIDR: "10.20.20.0/24", GatewayIPv4: "10.20.20.1", Purpose: "corporate"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !context.Reported || len(context.WANs) != 4 || len(context.Networks) != 1 {
		t.Fatalf("unexpected context: %#v", context)
	}

	byName := map[string]model.NetworkWAN{}
	for _, wan := range context.WANs {
		byName[wan.InterfaceName] = wan
	}
	if got := byName["eth8"]; got.AddressScope != AddressScopePublic || got.NATState != NATStateDirect || got.PublicIPMatch == nil || !*got.PublicIPMatch || got.PublicIPSource != PublicIPSourceServerPeer {
		t.Fatalf("direct WAN classified incorrectly: %#v", got)
	}
	if got := byName["eth9"]; got.AddressScope != AddressScopePrivate || got.NATState != NATStateDoubleNAT || !got.UpstreamNAT || !got.DoubleNAT || got.CGNAT {
		t.Fatalf("double-NAT WAN classified incorrectly: %#v", got)
	}
	if got := byName["wwan0"]; got.AddressScope != AddressScopeCGNAT || got.NATState != NATStateCGNAT || !got.UpstreamNAT || got.DoubleNAT || !got.CGNAT {
		t.Fatalf("CGNAT WAN classified incorrectly: %#v", got)
	}
	if got := byName["eth10"]; got.AddressScope != AddressScopePublic || got.NATState != NATStatePublicMismatch || got.PublicIPMatch == nil || *got.PublicIPMatch {
		t.Fatalf("public mismatch classified incorrectly: %#v", got)
	}
}

func TestReportReplacesCurrentSnapshotRowsAtomically(t *testing.T) {
	db, service, device := newNetworkContextService(t)
	vlan10 := 10
	if _, err := service.Report(device.ID, "8.8.8.8", ReportInput{
		WANs:     []WANInput{{InterfaceName: "wan0", Role: WANRolePrimary, DefaultRoute: true, IPv4: "8.8.8.8"}, {InterfaceName: "wan1", Role: WANRoleSecondary, IPv4: "192.168.1.2", PublicIPv4: "1.1.1.1"}},
		Networks: []SegmentInput{{Name: "OLD", VLANID: &vlan10, IPv4CIDR: "10.10.10.0/24", GatewayIPv4: "10.10.10.1"}},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Report(device.ID, "9.9.9.9", ReportInput{
		WANs:     []WANInput{{InterfaceName: "wan0", Role: WANRolePrimary, DefaultRoute: true, IPv4: "9.9.9.9"}},
		Networks: []SegmentInput{{Name: "NEW", IPv4CIDR: "10.20.30.0/24", GatewayIPv4: "10.20.30.1"}},
	}); err != nil {
		t.Fatal(err)
	}

	var wanCount, networkCount int64
	if err := db.Model(&model.NetworkWAN{}).Where("device_id = ?", device.ID).Count(&wanCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.NetworkSegment{}).Where("device_id = ?", device.ID).Count(&networkCount).Error; err != nil {
		t.Fatal(err)
	}
	if wanCount != 1 || networkCount != 1 {
		t.Fatalf("expected replacement snapshot rows, got WAN=%d networks=%d", wanCount, networkCount)
	}
	context, err := service.Context(device.ID)
	if err != nil {
		t.Fatal(err)
	}
	if context.Networks[0].Name != "NEW" || context.WANs[0].IPv4 != "9.9.9.9" {
		t.Fatalf("stale network context remained: %#v", context)
	}
}

func TestListContextsIncludesDevicesThatHaveNeverReported(t *testing.T) {
	db, service, device := newNetworkContextService(t)
	other := model.Device{Name: "ZZ-NO-NETWORK-CONTEXT", Status: "active"}
	if err := db.Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := service.Report(device.ID, "8.8.8.8", ReportInput{WANs: []WANInput{{InterfaceName: "wan0", Role: WANRolePrimary, DefaultRoute: true, IPv4: "8.8.8.8"}}}); err != nil {
		t.Fatal(err)
	}
	contexts, err := service.ListContexts()
	if err != nil {
		t.Fatal(err)
	}
	if len(contexts) != 2 || !contexts[0].Reported || contexts[1].Reported {
		t.Fatalf("unexpected fleet contexts: %#v", contexts)
	}
}

func TestValidationRejectsInvalidWANAndLANIdentity(t *testing.T) {
	_, service, device := newNetworkContextService(t)
	badVLAN := 4095
	cases := []struct {
		name  string
		input ReportInput
		err   error
	}{
		{"no WAN", ReportInput{}, ErrNoWAN},
		{"duplicate WAN", ReportInput{WANs: []WANInput{{InterfaceName: "WAN0"}, {InterfaceName: "wan0"}}}, ErrDuplicateInterface},
		{"multiple default routes", ReportInput{WANs: []WANInput{{InterfaceName: "wan0", DefaultRoute: true}, {InterfaceName: "wan1", DefaultRoute: true}}}, ErrMultipleDefaultRoutes},
		{"bad public IP", ReportInput{WANs: []WANInput{{InterfaceName: "wan0", PublicIPv4: "192.168.1.1"}}}, ErrInvalidPublicIPv4},
		{"bad vlan", ReportInput{WANs: []WANInput{{InterfaceName: "wan0"}}, Networks: []SegmentInput{{Name: "POS", VLANID: &badVLAN}}}, ErrInvalidVLAN},
		{"bad cidr", ReportInput{WANs: []WANInput{{InterfaceName: "wan0"}}, Networks: []SegmentInput{{Name: "POS", IPv4CIDR: "10.0.0.1"}}}, ErrInvalidCIDR},
		{"gateway outside", ReportInput{WANs: []WANInput{{InterfaceName: "wan0"}}, Networks: []SegmentInput{{Name: "POS", IPv4CIDR: "10.0.0.0/24", GatewayIPv4: "10.0.1.1"}}}, ErrInvalidGateway},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.Report(device.ID, "8.8.8.8", tc.input)
			if !errors.Is(err, tc.err) {
				t.Fatalf("expected %v, got %v", tc.err, err)
			}
		})
	}
}
