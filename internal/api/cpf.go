package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type ExternalCPFResponse struct {
	DADOS struct {
		CPF            string `json:"cpf"`
		Nome           string `json:"nome"`
		NomeMae        string `json:"nome_mae"`
		DataNascimento string `json:"data_nascimento"`
		Sexo           string `json:"sexo"`
	} `json:"DADOS"`
}

type CPFData struct {
	CPF       string `json:"cpf"`
	Nome      string `json:"nome"`
	NomeMae   string `json:"nome_mae"`
	Sexo      string `json:"sexo"`
	Situacao  string `json:"situacao"`
	DataNasc  string `json:"data_nasc"`
	Score     int    `json:"score"`
	LastCheck string `json:"last_check"`
}

func (s *Server) handleCPF(c *gin.Context) {
	cpf := c.Param("cpf")
	log.Printf("Handling CPF request for: %s", cpf)

	// Verificar configuração da API
	log.Printf("API Config - Endpoint: %s, Token: %s", s.config.API.CPFEndpoint, s.config.API.Token)

	// Verificar no cache primeiro
	var cpfData CPFData
	cachedData, err := s.cache.Get(cpf)
	if err != nil {
		log.Printf("Cache error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cache error"})
		return
	}

	if cachedData != "" {
		log.Printf("Cache hit for CPF: %s", cpf)
		if err := json.Unmarshal([]byte(cachedData), &cpfData); err != nil {
			log.Printf("Cache unmarshal error: %v", err)
			// Se houver erro ao decodificar o cache, vamos limpar e buscar novamente
			s.cache.Delete(cpf)
		} else if cpfData.CPF == "" || cpfData.Nome == "" {
			log.Printf("Invalid cached data for CPF: %s", cpf)
			s.cache.Delete(cpf)
		} else {
			s.metrics.IncrementCacheHits()
			c.JSON(http.StatusOK, cpfData)
			return
		}
	}

	s.metrics.IncrementCacheMisses()
	log.Printf("Cache miss for CPF: %s", cpf)

	// Fazer requisição para a API externa
	url := fmt.Sprintf("%s?cpf=%s&token=%s", s.config.API.CPFEndpoint, cpf, s.config.API.Token)
	log.Printf("Requesting external API: %s", url)

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("External API request error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "External API error"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("External API returned status code: %d", resp.StatusCode)
		c.JSON(resp.StatusCode, gin.H{"error": "External API error"})
		return
	}

	// Ler o corpo da resposta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error reading response"})
		return
	}
	log.Printf("External API response: %s", string(body))

	var externalResp ExternalCPFResponse
	if err := json.Unmarshal(body, &externalResp); err != nil {
		log.Printf("JSON unmarshal error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "JSON decode error"})
		return
	}

	log.Printf("External response parsed: %+v", externalResp)

	// Validar dados da resposta externa
	if externalResp.DADOS.CPF == "" || externalResp.DADOS.Nome == "" {
		log.Printf("Invalid external API response: missing required fields")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid CPF or data not found"})
		return
	}

	// Mapear a resposta externa para nosso formato
	cpfData = CPFData{
		CPF:       externalResp.DADOS.CPF,
		Nome:      externalResp.DADOS.Nome,
		NomeMae:   externalResp.DADOS.NomeMae,
		Sexo:      externalResp.DADOS.Sexo,
		Situacao:  "REGULAR", // Valor padrão
		DataNasc:  externalResp.DADOS.DataNascimento,
		Score:     0, // Valor padrão
		LastCheck: time.Now().Format("2006-01-02 15:04:05"),
	}

	log.Printf("Mapped data: %+v", cpfData)

	// Salvar no cache
	jsonData, err := json.Marshal(cpfData)
	if err != nil {
		log.Printf("JSON marshal error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "JSON encode error"})
		return
	}

	if err := s.cache.Set(cpf, string(jsonData), 24*time.Hour); err != nil {
		log.Printf("Cache set error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cache save error"})
		return
	}

	c.JSON(http.StatusOK, cpfData)
} 