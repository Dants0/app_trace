package main

import (
	"app_trace/internal/ai"
	"app_trace/internal/parser"
	"bufio"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Erro ao carregar o arquivo .env")
	}

	// Pega a chave da variável de ambiente
	apiKey := os.Getenv("OPENAI_API_KEY")
	port := os.Getenv("PORT")

	r := gin.Default()

	r.POST("/process-batch", func(c *gin.Context) {
		form, err := c.MultipartForm()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Falha ao ler arquivos"})
			return
		}

		files := form.File["files"]
		var wg sync.WaitGroup
		resultsChan := make(chan gin.H, len(files))

		for _, fileHeader := range files {
			wg.Add(1)
			go func(header *multipart.FileHeader) {
				defer wg.Done()

				f, err := header.Open()
				if err != nil {
					return
				}
				defer f.Close()

				// Scanner com buffer robusto para queries gigantes
				scanner := bufio.NewScanner(f)
				buf := make([]byte, 0, 5*1024*1024) // 5MB
				scanner.Buffer(buf, 5*1024*1024)

				events := parser.ParseAndDeduplicate(scanner)

				// Verificação de erro do scanner (opcional logar no console)
				if scanner.Err() != nil {
					// fmt.Printf("Erro no scanner para %s: %v\n", header.Filename, scanner.Err())
				}

				resultsChan <- gin.H{
					"filename":    header.Filename,
					"event_count": len(events),
					"data":        events,
				}
			}(fileHeader)
		}

		go func() {
			wg.Wait()
			close(resultsChan)
		}()

		var finalResponse []gin.H
		for res := range resultsChan {
			finalResponse = append(finalResponse, res)
		}

		c.JSON(http.StatusOK, finalResponse)
	})

	r.POST("/analyze-ai", func(c *gin.Context) {
		form, err := c.MultipartForm()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Falha ao ler ficheiros"})
			return
		}

		files := form.File["files"]
		var wg sync.WaitGroup
		resultsChan := make(chan gin.H, len(files))

		for _, fileHeader := range files {
			wg.Add(1)
			go func(header *multipart.FileHeader) {
				defer wg.Done()

				f, _ := header.Open()
				defer f.Close()

				scanner := bufio.NewScanner(f)
				buf := make([]byte, 0, 5*1024*1024)
				scanner.Buffer(buf, 5*1024*1024)

				events := parser.ParseAndDeduplicate(scanner)

				strategicPlan, err := ai.GetStrategicPlan(events, apiKey)
				if err != nil {
					strategicPlan = "Falha ao gerar análise: " + err.Error()
				}

				resultsChan <- gin.H{
					"filename":        header.Filename,
					"event_count":     len(events),
					"strategic_plan":  strategicPlan,
					"structured_data": events,
				}
			}(fileHeader)
		}

		go func() {
			wg.Wait()
			close(resultsChan)
		}()

		var finalResponse []gin.H
		for res := range resultsChan {
			finalResponse = append(finalResponse, res)
		}

		c.JSON(http.StatusOK, finalResponse)
	})

	r.POST("/analyze-trace", func(c *gin.Context) {
		form, err := c.MultipartForm()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Falha ao ler arquivos"})
			return
		}

		files := form.File["files"]
		var wg sync.WaitGroup
		resultsChan := make(chan gin.H, len(files))

		for _, fileHeader := range files {
			wg.Add(1)
			go func(header *multipart.FileHeader) {
				defer wg.Done()

				f, err := header.Open()
				if err != nil {
					return
				}
				defer f.Close()

				// 1. Motor de Normalização e Deduplicação
				scanner := bufio.NewScanner(f)
				buf := make([]byte, 0, 5*1024*1024)
				scanner.Buffer(buf, 5*1024*1024)

				events := parser.ParseAndDeduplicate(scanner)

				// 2. Análise Estratégica via IA (Thread de Resolução)
				strategicPlan, err := ai.GetStrategicPlan(events, apiKey)
				if err != nil {
					strategicPlan = "Erro na análise: " + err.Error()
				}

				
				resultsChan <- gin.H{
					"filename": header.Filename,
					"summary": gin.H{
						"total_events":   len(events),
						"critical_count": parser.CountBySeverity(events, "CRITICAL"),
					},
					"strategic_analysis": strategicPlan,
					"structured_data":    events,
				}
			}(fileHeader)
		}

		go func() {
			wg.Wait()
			close(resultsChan)
		}()

		var finalResponse []gin.H
		for res := range resultsChan {
			finalResponse = append(finalResponse, res)
		}

		c.JSON(http.StatusOK, finalResponse)
	})

	r.Run(port)
}
