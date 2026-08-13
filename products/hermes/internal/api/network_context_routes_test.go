package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mca-rolando/HermesDDNS/internal/model"
	"github.com/mca-rolando/HermesDDNS/internal/networkcontext"
)

func TestAgentNetworkContextRequiresIdentityAndClassifiesDoubleNAT(t *testing.T) {
	db, server, device, _ := newCredentialAPITestServer(t)
	_, agentKey, err := server.AgentAuth.IssueCredential(device.ID)
	if err != nil {
		t.Fatal(err)
	}

	unauthorizedReq := httptest.NewRequest(http.MethodPost, "/api/v1/agent/network-context", bytes.NewBufferString(`{}`))
	unauthorizedReq.Header.Set("Content-Type", "application/json")
	unauthorizedRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(unauthorizedRec, unauthorizedReq)
	if unauthorizedRec.Code != http.StatusUnauthorized {
		t.Fatalf("network context without identity expected %d, got %d", http.StatusUnauthorized, unauthorizedRec.Code)
	}

	body := bytes.NewBufferString(`{
		"wans":[
			{"interface_name":"eth8","role":"primary","default_route":true,"ipv4":"192.168.1.20","gateway_ipv4":"192.168.1.1"},
			{"interface_name":"eth9","role":"secondary","ipv4":"100.72.35.18","public_ipv4":"8.8.8.8"}
		],
		"networks":[
			{"name":"LAN","ipv4_cidr":"10.222.0.0/24","gateway_ipv4":"10.222.0.1","purpose":"corporate"},
			{"name":"POS","vlan_id":20,"ipv4_cidr":"10.222.20.0/24","gateway_ipv4":"10.222.20.1","purpose":"corporate"}
		]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent/network-context", body)
	req.Header.Set("Authorization", "Bearer "+agentKey.Plaintext)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "73.44.18.91:57000"
	rec := httptest.NewRecorder()
	server.Echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("network context expected %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "secret_hash") || strings.Contains(rec.Body.String(), agentKey.Plaintext) {
		t.Fatalf("network context response must not expose credentials: %s", rec.Body.String())
	}

	var response networkcontext.DeviceNetworkContext
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Device.ID != device.ID || !response.Reported || len(response.WANs) != 2 || len(response.Networks) != 2 {
		t.Fatalf("unexpected network context response: %#v", response)
	}
	if response.WANs[0].InterfaceName != "eth8" || response.WANs[0].AddressScope != networkcontext.AddressScopePrivate || response.WANs[0].NATState != networkcontext.NATStateDoubleNAT || !response.WANs[0].DoubleNAT {
		t.Fatalf("primary WAN double NAT not detected: %#v", response.WANs[0])
	}
	if response.WANs[1].AddressScope != networkcontext.AddressScopeCGNAT || !response.WANs[1].CGNAT {
		t.Fatalf("secondary WAN CGNAT not detected: %#v", response.WANs[1])
	}

	var storedSnapshot model.NetworkIdentitySnapshot
	if err := db.Where("device_id = ?", device.ID).First(&storedSnapshot).Error; err != nil {
		t.Fatal(err)
	}
	if storedSnapshot.ServerObservedIP != "73.44.18.91" {
		t.Fatalf("server-observed IP not stored: %#v", storedSnapshot)
	}
}

func TestAgentNetworkContextCannotSelectAnotherDevice(t *testing.T) {
	db, server, device, _ := newCredentialAPITestServer(t)
	_, agentKey, err := server.AgentAuth.IssueCredential(device.ID)
	if err != nil {
		t.Fatal(err)
	}
	other := model.Device{Name: "OTHER-NETWORK", Status: "active"}
	if err := db.Create(&other).Error; err != nil {
		t.Fatal(err)
	}

	body := bytes.NewBufferString(fmt.Sprintf(`{"device_id":%d,"wans":[{"interface_name":"wan0","role":"primary","default_route":true,"ipv4":"8.8.8.8"}]}`, other.ID))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent/network-context", body)
	req.Header.Set("Authorization", "Bearer "+agentKey.Plaintext)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "8.8.8.8:50000"
	rec := httptest.NewRecorder()
	server.Echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("network context expected %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var ownCount, otherCount int64
	if err := db.Model(&model.NetworkIdentitySnapshot{}).Where("device_id = ?", device.ID).Count(&ownCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.NetworkIdentitySnapshot{}).Where("device_id = ?", other.ID).Count(&otherCount).Error; err != nil {
		t.Fatal(err)
	}
	if ownCount != 1 || otherCount != 0 {
		t.Fatalf("network context identity was not bound to authenticated Device: own=%d other=%d", ownCount, otherCount)
	}
}

func TestNetworkContextAdminAPISupportsDeviceAndFleetViews(t *testing.T) {
	db, server, device, _ := newCredentialAPITestServer(t)
	other := model.Device{Name: "ZZ-NO-CONTEXT", Status: "active"}
	if err := db.Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	_, agentKey, err := server.AgentAuth.IssueCredential(device.ID)
	if err != nil {
		t.Fatal(err)
	}

	reportReq := httptest.NewRequest(http.MethodPost, "/api/v1/agent/network-context", bytes.NewBufferString(`{"wans":[{"interface_name":"wan0","role":"primary","default_route":true,"ipv4":"8.8.8.8"}],"networks":[{"name":"LAN","ipv4_cidr":"10.0.0.0/24","gateway_ipv4":"10.0.0.1"}]}`))
	reportReq.Header.Set("Authorization", "Bearer "+agentKey.Plaintext)
	reportReq.Header.Set("Content-Type", "application/json")
	reportReq.RemoteAddr = "8.8.8.8:50000"
	reportRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(reportRec, reportReq)
	if reportRec.Code != http.StatusOK {
		t.Fatalf("report expected %d, got %d: %s", http.StatusOK, reportRec.Code, reportRec.Body.String())
	}

	deviceReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/devices/%d/network-context", device.ID), nil)
	deviceRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(deviceRec, deviceReq)
	if deviceRec.Code != http.StatusOK || !strings.Contains(deviceRec.Body.String(), `"reported":true`) {
		t.Fatalf("device network context unexpected: %d %s", deviceRec.Code, deviceRec.Body.String())
	}

	neverReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/devices/%d/network-context", other.ID), nil)
	neverRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(neverRec, neverReq)
	if neverRec.Code != http.StatusOK || !strings.Contains(neverRec.Body.String(), `"reported":false`) {
		t.Fatalf("unreported device context unexpected: %d %s", neverRec.Code, neverRec.Body.String())
	}

	fleetReq := httptest.NewRequest(http.MethodGet, "/api/v1/network-context", nil)
	fleetRec := httptest.NewRecorder()
	server.Echo.ServeHTTP(fleetRec, fleetReq)
	if fleetRec.Code != http.StatusOK {
		t.Fatalf("fleet network context expected %d, got %d: %s", http.StatusOK, fleetRec.Code, fleetRec.Body.String())
	}
	var fleet []networkcontext.DeviceNetworkContext
	if err := json.Unmarshal(fleetRec.Body.Bytes(), &fleet); err != nil {
		t.Fatal(err)
	}
	if len(fleet) != 2 || !fleet[0].Reported || fleet[1].Reported {
		t.Fatalf("unexpected fleet network context: %#v", fleet)
	}
}

func TestAgentNetworkContextRejectsInvalidTopologyData(t *testing.T) {
	_, server, device, _ := newCredentialAPITestServer(t)
	_, agentKey, err := server.AgentAuth.IssueCredential(device.ID)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent/network-context", bytes.NewBufferString(`{"wans":[{"interface_name":"wan0","role":"primary","default_route":true,"ipv4":"8.8.8.8"}],"networks":[{"name":"BAD","vlan_id":5000,"ipv4_cidr":"10.0.0.0/24","gateway_ipv4":"10.0.1.1"}]}`))
	req.Header.Set("Authorization", "Bearer "+agentKey.Plaintext)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.Echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid network context expected %d, got %d: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}
