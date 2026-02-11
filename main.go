package main

import (
	"app_trace/internal/parser"
	"bufio"
	"mime/multipart"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

func main() {
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

	r.Run(":8000")
}