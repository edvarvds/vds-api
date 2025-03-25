package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type CPFData struct {
	CPF       string `json:"cpf"`
	Nome      string `json:"nome"`
	Situacao  string `json:"situacao"`
	DataNasc  string `json:"data_nasc"`
	Score     int    `json:"score"`
	LastCheck string `json:"last_check"`
}

func (s *Server) handleCPF(c *gin.Context) {
	cpf := c.Param("cpf")

	// Verificar no cache primeiro
	var cpfData CPFData
	cachedData, err := s.cache.Get(cpf)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cache error"})
		return
	}

	if cachedData != "" {
		if err := json.Unmarshal([]byte(cachedData), &cpfData); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Cache data error"})
			return
		}
		s.metrics.IncrementCacheHits()
		c.JSON(http.StatusOK, cpfData)
		return
	}

	s.metrics.IncrementCacheMisses()

	// Fazer requisição para a API externa
	resp, err := http.Get(fmt.Sprintf("https://consulta.fontesderenda.blog/cpf.php?cpf=%s&token=6285fe45-e991-4071-a848-3fac8273c82a", cpf))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "External API error"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(resp.StatusCode, gin.H{"error": "External API error"})
		return
	}

	if err := json.NewDecoder(resp.Body).Decode(&cpfData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "JSON decode error"})
		return
	}

	// Salvar no cache
	jsonData, err := json.Marshal(cpfData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "JSON encode error"})
		return
	}

	if err := s.cache.Set(cpf, string(jsonData), 24*time.Hour); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cache save error"})
		return
	}

	c.JSON(http.StatusOK, cpfData)
} 