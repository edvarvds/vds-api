package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type PendingDomain struct {
	Domain       string    `json:"domain"`
	RequestCount int64     `json:"request_count"`
	FirstRequest time.Time `json:"first_request"`
	LastRequest  time.Time `json:"last_request"`
}

type Metrics struct {
	TotalRequests   int64                    `json:"total_requests"`
	CacheHits       int64                    `json:"cache_hits"`
	CacheMisses     int64                    `json:"cache_misses"`
	RateLimitBlocks int64                    `json:"rate_limit_blocks"`
	PendingDomains  map[string]*PendingDomain `json:"pending_domains"`
}

func NewMetrics() *Metrics {
	return &Metrics{
		PendingDomains: make(map[string]*PendingDomain),
	}
}

func (m *Metrics) IncrementRequests() {
	m.TotalRequests++
}

func (m *Metrics) IncrementCacheHits() {
	m.CacheHits++
}

func (m *Metrics) IncrementCacheMisses() {
	m.CacheMisses++
}

func (m *Metrics) IncrementRateLimitBlock() {
	m.RateLimitBlocks++
}

func (m *Metrics) AddPendingDomain(domain string) {
	if _, exists := m.PendingDomains[domain]; !exists {
		now := time.Now()
		m.PendingDomains[domain] = &PendingDomain{
			Domain:       domain,
			RequestCount: 1,
			FirstRequest: now,
			LastRequest:  now,
		}
	} else {
		m.PendingDomains[domain].RequestCount++
		m.PendingDomains[domain].LastRequest = time.Now()
	}
}

func (m *Metrics) GetMetrics() *Metrics {
	return m
}

func (s *Server) setupAdminRoutes() {
	admin := s.router.Group("/admin")
	admin.Use(s.adminAuthMiddleware())
	{
		admin.GET("/", s.handleAdminPanel)
		admin.GET("/metrics", s.handleMetrics)
		admin.POST("/domains", s.handleAddDomain)
		admin.DELETE("/domains/:domain", s.handleDeleteDomain)
		admin.POST("/domains/approve/:domain", s.handleApproveDomain)
		admin.POST("/cache/clear", s.handleClearCache)
	}
}

func (s *Server) adminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("X-Admin-Token")
		if token != s.config.Server.AdminToken {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func (s *Server) handleAdminPanel(c *gin.Context) {
	c.HTML(http.StatusOK, "admin.html", gin.H{
		"metrics":        s.metrics.GetMetrics(),
		"AllowedHosts":   s.config.Server.AllowedHosts,
		"PendingDomains": s.metrics.PendingDomains,
	})
}

func (s *Server) handleMetrics(c *gin.Context) {
	c.JSON(http.StatusOK, s.metrics.GetMetrics())
}

func (s *Server) handleAddDomain(c *gin.Context) {
	var req struct {
		Domain string `json:"domain"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verificar se o domínio já está na lista
	for _, domain := range s.config.Server.AllowedHosts {
		if domain == req.Domain {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Domain already exists"})
			return
		}
	}

	s.config.Server.AllowedHosts = append(s.config.Server.AllowedHosts, req.Domain)
	c.JSON(http.StatusOK, gin.H{"message": "Domain added successfully"})
}

func (s *Server) handleDeleteDomain(c *gin.Context) {
	domain := c.Param("domain")
	for i, d := range s.config.Server.AllowedHosts {
		if d == domain {
			s.config.Server.AllowedHosts = append(s.config.Server.AllowedHosts[:i], s.config.Server.AllowedHosts[i+1:]...)
			c.JSON(http.StatusOK, gin.H{"message": "Domain deleted successfully"})
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "Domain not found"})
}

func (s *Server) handleApproveDomain(c *gin.Context) {
	domain := c.Param("domain")
	
	// Verificar se o domínio está na lista de pendentes
	if _, exists := s.metrics.PendingDomains[domain]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Domain not found in pending list"})
		return
	}

	// Verificar se o domínio já está na lista de aprovados
	for _, d := range s.config.Server.AllowedHosts {
		if d == domain {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Domain already approved"})
			return
		}
	}

	// Adicionar o domínio à lista de aprovados
	s.config.Server.AllowedHosts = append(s.config.Server.AllowedHosts, domain)
	
	// Remover o domínio da lista de pendentes
	delete(s.metrics.PendingDomains, domain)

	c.JSON(http.StatusOK, gin.H{"message": "Domain approved successfully"})
}

func (s *Server) handleClearCache(c *gin.Context) {
	cpf := c.Query("cpf")
	if cpf == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CPF parameter is required"})
		return
	}

	if err := s.cache.Delete(cpf); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error clearing cache: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cache cleared successfully"})
} 