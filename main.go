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

	"github.com/gin-contrib/cors" 
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Aviso: .env não encontrado, usando variáveis de ambiente do sistema")
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	port := os.Getenv("PORT")

	r := gin.Default()

	// CONFIGURAÇÃO DE CORS CORRIGIDA
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"POST", "GET", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	r.POST("/analyze-trace", func(c *gin.Context) {
		bugDescription := c.PostForm("bugDescription")
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

				scanner := bufio.NewScanner(f)
				buf := make([]byte, 0, 5*1024*1024)
				scanner.Buffer(buf, 5*1024*1024)

				events := parser.ParseAndDeduplicate(scanner)

				// IA recebe o relato do bug + eventos estruturados
				strategicPlan, err := ai.GetStrategicPlan(events, apiKey, bugDescription)
				if err != nil {
					strategicPlan = "Erro na análise: " + err.Error()
				}

				resultsChan <- gin.H{
					"filename":           header.Filename,
					"event_count":        len(events),
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

	r.POST("/process-batch", func(c *gin.Context) {
		form, _ := c.MultipartForm()
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
				events := parser.ParseAndDeduplicate(scanner)
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
		bugDescription := c.PostForm("bugDescription")

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

				strategicPlan, err := ai.GetStrategicPlan(events, apiKey, bugDescription)
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

	r.Run(port)
}
