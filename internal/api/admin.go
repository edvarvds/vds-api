package api

import (
	"fmt"
	"net/http"
	"sync"
	"time"
	"sync/atomic"

	"github.com/gin-gonic/gin"
)

type PendingDomain struct {
	Domain       string    `json:"domain"`
	RequestCount int64     `json:"request_count"`
	FirstRequest time.Time `json:"first_request"`
	LastRequest  time.Time `json:"last_request"`
}

type Metrics struct {
	mu              sync.RWMutex
	Requests        int64
	RateLimitBlocks int64
	BlockedIPs      int64
	CacheHits       int64
	CacheMisses     int64
	PendingDomains  map[string]int64
}

func NewMetrics() *Metrics {
	return &Metrics{
		PendingDomains: make(map[string]int64),
	}
}

func (m *Metrics) IncrementRequests() {
	atomic.AddInt64(&m.Requests, 1)
}

func (m *Metrics) IncrementRateLimitBlock() {
	atomic.AddInt64(&m.RateLimitBlocks, 1)
}

func (m *Metrics) IncrementBlockedIPs() {
	atomic.AddInt64(&m.BlockedIPs, 1)
}

func (m *Metrics) IncrementCacheHits() {
	atomic.AddInt64(&m.CacheHits, 1)
}

func (m *Metrics) IncrementCacheMisses() {
	atomic.AddInt64(&m.CacheMisses, 1)
}

func (m *Metrics) AddPendingDomain(domain string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PendingDomains[domain]++
}

func (m *Metrics) GetMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"total_requests": atomic.LoadInt64(&m.Requests),
		"rate_limit_blocks": atomic.LoadInt64(&m.RateLimitBlocks),
		"blocked_ips": atomic.LoadInt64(&m.BlockedIPs),
		"cache_hits": atomic.LoadInt64(&m.CacheHits),
		"cache_misses": atomic.LoadInt64(&m.CacheMisses),
		"pending_domains": m.PendingDomains,
	}
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