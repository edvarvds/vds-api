package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
	"api_vds/internal/config"
	"api_vds/internal/cache"
)

type Server struct {
	router      *gin.Engine
	config      *config.Config
	cache       *cache.RedisClient
	rateLimiters map[string]*rate.Limiter
	metrics     *Metrics
	httpServer  *http.Server
}

func NewServer(cfg *config.Config, cache *cache.RedisClient) *Server {
	server := &Server{
		router:      gin.Default(),
		config:      cfg,
		cache:       cache,
		rateLimiters: make(map[string]*rate.Limiter),
		metrics:     NewMetrics(),
	}

	// Load templates
	server.router.LoadHTMLGlob("templates/*")

	server.setupMiddleware()
	server.setupRoutes()
	server.setupAdminRoutes()

	server.httpServer = &http.Server{
		Addr:    cfg.Server.Port,
		Handler: server.router,
	}

	return server
}

func (s *Server) getRateLimiter(ip string) *rate.Limiter {
	if limiter, exists := s.rateLimiters[ip]; exists {
		return limiter
	}
	limiter := rate.NewLimiter(rate.Every(time.Second), s.config.Server.RateLimit)
	s.rateLimiters[ip] = limiter
	return limiter
}

func (s *Server) setupMiddleware() {
	// IP validation middleware
	s.router.Use(func(c *gin.Context) {
		clientIP := c.ClientIP()
		if !s.isAllowedIP(clientIP) {
			s.metrics.IncrementBlockedIPs()
			c.JSON(http.StatusForbidden, gin.H{
				"error": "IP not allowed",
				"message": "Your IP is not authorized to access this API.",
			})
			c.Abort()
			return
		}
		c.Next()
	})

	// CORS middleware
	s.router.Use(func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		// Bloquear requisições sem Origin header
		if origin == "" {
			s.metrics.IncrementBlockedIPs()
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Origin required",
				"message": "Direct API access is not allowed. Please use a web browser or make requests from an allowed domain.",
			})
			c.Abort()
			return
		}

		if !s.isAllowedOrigin(origin) {
			s.metrics.AddPendingDomain(origin)
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Origin not allowed",
				"message": "Your domain is pending approval. Please contact the administrator.",
			})
			c.Abort()
			return
		}

		// Adicionar headers CORS
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		c.Next()
	})

	// Rate limiting middleware (apenas para rotas da API)
	s.router.Use(func(c *gin.Context) {
		// Não aplicar rate limit para rotas administrativas
		if strings.HasPrefix(c.Request.URL.Path, "/admin") {
			c.Next()
			return
		}

		ip := c.ClientIP()
		limiter := s.getRateLimiter(ip)
		
		if !limiter.Allow() {
			s.metrics.IncrementRateLimitBlock()
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
			c.Abort()
			return
		}
		c.Next()
	})

	// Metrics middleware (apenas para rotas da API)
	s.router.Use(func(c *gin.Context) {
		// Não contar métricas para rotas administrativas
		if strings.HasPrefix(c.Request.URL.Path, "/admin") {
			c.Next()
			return
		}
		
		s.metrics.IncrementRequests()
		c.Next()
	})
}

func (s *Server) setupRoutes() {
	api := s.router.Group("/api/v1")
	{
		api.GET("/cpf/:cpf", s.handleCPF)
	}
}

func (s *Server) isAllowedOrigin(origin string) bool {
	if len(s.config.Server.AllowedHosts) == 0 || s.config.Server.AllowedHosts[0] == "*" {
		return true
	}

	for _, allowed := range s.config.Server.AllowedHosts {
		if strings.EqualFold(origin, allowed) {
			return true
		}
	}
	return false
}

func (s *Server) isAllowedIP(ip string) bool {
	if len(s.config.Server.AllowedIPs) == 0 {
		return false
	}

	for _, allowed := range s.config.Server.AllowedIPs {
		if ip == allowed {
			return true
		}
	}
	return false
}

func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
} 