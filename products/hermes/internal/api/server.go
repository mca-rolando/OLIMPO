package api

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/mca-rolando/HermesDDNS/internal/buildinfo"
	"github.com/mca-rolando/HermesDDNS/internal/config"
	"github.com/mca-rolando/HermesDDNS/internal/credential"
	"github.com/mca-rolando/HermesDDNS/internal/ddns"
	"github.com/mca-rolando/HermesDDNS/internal/model"
	"github.com/mca-rolando/HermesDDNS/internal/security"
	"github.com/tg123/go-htpasswd"
	"gorm.io/gorm"
)

type validatorAdapter struct{ v *validator.Validate }

func (v validatorAdapter) Validate(i interface{}) error { return v.v.Struct(i) }

type Server struct {
	Echo        *echo.Echo
	DB          *gorm.DB
	Config      config.Config
	DDNS        *ddns.Service
	Credentials *credential.Service
}

func New(db *gorm.DB, cfg config.Config, ddnsService *ddns.Service) *Server {
	e := echo.New()
	e.HideBanner = true
	e.Validator = validatorAdapter{v: validator.New()}
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.Logger())

	s := &Server{Echo: e, DB: db, Config: cfg, DDNS: ddnsService, Credentials: &credential.Service{DB: db}}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.Echo.GET("/health", s.health)
	s.Echo.GET("/ping", s.health)

	for _, path := range []string{"/update", "/nic/update", "/v2/update", "/v3/update"} {
		s.Echo.GET(path, s.ddnsUpdate)
	}

	api := s.Echo.Group(s.Config.APIPrefix)
	api.GET("/health", s.health)
	api.GET("/system/version", func(c echo.Context) error { return c.JSON(http.StatusOK, buildinfo.Current()) })

	admin := api.Group("")
	if s.Config.AdminLogin != "" {
		admin.Use(middleware.BasicAuth(s.authenticateAdmin))
	}
	admin.GET("/domains", s.listDomains)
	admin.POST("/domains", s.createDomain)
	admin.GET("/devices", s.listDevices)
	admin.POST("/devices", s.createDevice)
	admin.GET("/devices/:id/credentials", s.listCredentials)
	admin.POST("/devices/:id/credentials/rotate", s.requestCredentialRotation)
	admin.POST("/devices/:id/credential-rotations", s.requestCredentialRotation)
	admin.GET("/devices/:id/credential-rotations", s.listCredentialRotations)
	admin.GET("/devices/:id/credential-rotations/:rotation_id", s.getCredentialRotation)
	admin.POST("/devices/:id/credential-rotations/:rotation_id/rollback", s.rollbackCredentialRotation)
	admin.POST("/credential-rotations/reconcile", s.reconcileCredentialRotations)
	admin.GET("/logs", s.listLogs)
}

func (s *Server) health(c echo.Context) error {
	sqlDB, err := s.DB.DB()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]any{"status": "degraded", "database": "error"})
	}
	if err := sqlDB.Ping(); err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]any{"status": "degraded", "database": "error"})
	}
	return c.JSON(http.StatusOK, map[string]any{"status": "ok", "database": "ok", "build": buildinfo.Current()})
}

func (s *Server) authenticateAdmin(username, password string, _ echo.Context) (bool, error) {
	pw, err := htpasswd.NewFromReader(strings.NewReader(s.Config.AdminLogin), htpasswd.DefaultSystems, nil)
	if err != nil {
		return false, err
	}
	return pw.Match(username, password), nil
}

func (s *Server) ddnsUpdate(c echo.Context) error {
	username, password, ok := c.Request().BasicAuth()
	if !ok {
		return c.String(http.StatusUnauthorized, "badauth\n")
	}
	callerIP := s.callerIP(c)
	auth, err := s.DDNS.Authenticate(username, password, callerIP)
	if err != nil {
		return c.String(http.StatusUnauthorized, "badauth\n")
	}

	result, err := s.DDNS.Update(auth, c.QueryParam("hostname"), c.QueryParam("myip"), callerIP, c.Request().UserAgent())
	if err != nil {
		switch {
		case errors.Is(err, ddns.ErrBadAuth):
			return c.String(http.StatusUnauthorized, "badauth\n")
		case errors.Is(err, ddns.ErrNotFQDN):
			return c.String(http.StatusBadRequest, "notfqdn\n")
		case errors.Is(err, ddns.ErrBadIP):
			return c.String(http.StatusBadRequest, "badagent\n")
		default:
			return c.String(http.StatusServiceUnavailable, "dnserr\n")
		}
	}
	if s.Credentials != nil {
		if _, err := s.Credentials.ConfirmCredentialUse(auth.Credential.ID); err != nil {
			s.Echo.Logger.Errorf("confirm DDNS credential use: %v", err)
		}
	}
	return c.String(http.StatusOK, fmt.Sprintf("%s %s\n", result.Code, result.IP))
}

func (s *Server) callerIP(c echo.Context) string {
	if s.Config.TrustProxyHeaders {
		if xri := strings.TrimSpace(c.Request().Header.Get("X-Real-IP")); xri != "" {
			return xri
		}
		if xff := strings.TrimSpace(c.Request().Header.Get("X-Forwarded-For")); xff != "" {
			return strings.TrimSpace(strings.Split(xff, ",")[0])
		}
	}
	host, _, err := net.SplitHostPort(c.Request().RemoteAddr)
	if err == nil {
		return host
	}
	return c.Request().RemoteAddr
}

type createDomainRequest struct {
	Name       string `json:"name" validate:"required,fqdn"`
	DefaultTTL int    `json:"default_ttl" validate:"omitempty,min=20,max=86400"`
	Wildcard   bool   `json:"wildcard"`
}

func (s *Server) createDomain(c echo.Context) error {
	var req createDomainRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	name := strings.TrimSuffix(strings.ToLower(req.Name), ".")
	if !contains(s.Config.AllowedDomains, name) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "domain is not listed in HERMES_DOMAINS"})
	}
	if req.DefaultTTL == 0 {
		req.DefaultTTL = s.Config.DefaultTTL
	}
	d := model.Domain{Name: name, DefaultTTL: req.DefaultTTL, Enabled: true, Wildcard: req.Wildcard}
	if err := s.DB.Create(&d).Error; err != nil {
		return c.JSON(http.StatusConflict, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, d)
}
func (s *Server) listDomains(c echo.Context) error {
	var domains []model.Domain
	if err := s.DB.Order("name asc").Find(&domains).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, domains)
}

type createDeviceRequest struct {
	Name        string `json:"name" validate:"required,min=3,max=128"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
}

func (s *Server) createDevice(c echo.Context) error {
	var req createDeviceRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	dev := model.Device{Name: req.Name, DisplayName: req.DisplayName, Type: req.Type, Status: "active"}
	key, err := security.GenerateDDNSKey()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	now := time.Now().UTC()

	err = s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&dev).Error; err != nil {
			return err
		}
		cred := model.DDNSCredential{DeviceID: dev.ID, KeyID: key.ID, SecretHash: key.Hash, Status: model.CredentialStatusActive, ActivatedAt: &now}
		return tx.Create(&cred).Error
	})
	if err != nil {
		return c.JSON(http.StatusConflict, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, map[string]any{"device": dev, "ddns_username": dev.Name, "api_key": key.Plaintext, "warning": "API key is returned once. Store it securely."})
}
func (s *Server) listDevices(c echo.Context) error {
	var devices []model.Device
	if err := s.DB.Order("name asc").Find(&devices).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, devices)
}
func (s *Server) listCredentials(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid device id"})
	}
	var creds []model.DDNSCredential
	if err := s.DB.Where("device_id = ?", uint(id)).Order("created_at desc").Find(&creds).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, creds)
}

func (s *Server) listLogs(c echo.Context) error {
	var logs []model.UpdateLog
	if err := s.DB.Order("created_at desc").Limit(100).Find(&logs).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, logs)
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if strings.EqualFold(item, target) {
			return true
		}
	}
	return false
}
