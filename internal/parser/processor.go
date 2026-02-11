package parser

import (
	"app_trace/internal/models"
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"regexp"
	"strconv"
	"strings"
)

var headerRegex = regexp.MustCompile(``)
var numberRegex = regexp.MustCompile(`\b\d+\b`)

func ParseAndDeduplicate(scanner *bufio.Scanner) []models.LogEvent {
	aggregated := make(map[string]*models.LogEvent)
	var order []string
	var currentRawEvent *models.LogEvent
	
	noise := []string{"Length=", "BUFFER", "Bind Columns", "COLUMNS SELECTED", "VCHAR", "CHAR"}

	for scanner.Scan() {
		line := scanner.Text()
		
		// Limpeza bruta de caracteres que quebram o UTF-8 ou o parse
		line = strings.Map(func(r rune) rune {
			if r < 32 && r != '\n' && r != '\t' { return -1 } // Remove caracteres de controle e null bytes
			return r
		}, line)
		
		line = strings.TrimSpace(line)
		if line == "" || containsNoise(line, noise) {
			continue
		}

		// A assinatura de uma linha de trace é sempre o parêntese com "MS / "
		// Ex: (0.002 MS / 78282.558 MS)
		if strings.Contains(line, " MS /") && strings.Contains(line, " MS)") {
			
			if currentRawEvent != nil {
				processAggregation(aggregated, &order, currentRawEvent)
			}

			// 1. Extrair Sessão: Tudo entre o primeiro "(" e o primeiro ")"
			openParen := strings.Index(line, "(")
			closeParen := strings.Index(line, ")")
			
			session := "unknown"
			if openParen != -1 && closeParen > openParen {
				session = strings.TrimSpace(line[openParen+1 : closeParen])
			}

			// 2. Extrair Ação: O que está entre o fechamento da sessão e a abertura do tempo
			// (04547C10): PREPARE: (0.000 MS / 614.409 MS)
			lastOpenParen := strings.LastIndex(line, "(")
			actionPart := ""
			if closeParen != -1 && lastOpenParen > closeParen {
				actionPart = line[closeParen+1 : lastOpenParen]
				actionPart = strings.Trim(actionPart, " :") // Remove espaços e dois pontos
			}

			// 3. Extrair Delta: Dentro do último parêntese
			delta := 0.0
			timePart := line[lastOpenParen+1:]
			slashIdx := strings.Index(timePart, "/")
			if slashIdx != -1 {
				deltaStr := strings.TrimSpace(strings.ReplaceAll(timePart[:slashIdx], "MS", ""))
				delta, _ = strconv.ParseFloat(deltaStr, 64)
			}

			currentRawEvent = &models.LogEvent{
				Session:      session,
				Action:       actionPart,
				Count:        1,
				TotalDeltaMS: delta,
			}

		} else if currentRawEvent != nil && !strings.HasPrefix(line, "(") && !strings.HasPrefix(line, "/*") {
			// Acúmulo de SQL (linhas que não são rastro nem comentário)
			if currentRawEvent.Statement == "" {
				currentRawEvent.Statement = line
			} else {
				currentRawEvent.Statement += " " + line
			}
		}
	}

	if currentRawEvent != nil {
		processAggregation(aggregated, &order, currentRawEvent)
	}

	return finalize(aggregated, order)
}

func processAggregation(m map[string]*models.LogEvent, order *[]string, e *models.LogEvent) {
	// Template Mining: normaliza números para permitir agrupamento de patterns
	norm := numberRegex.ReplaceAllString(e.Statement, "?")
	hash := generateHash(e.Action + norm)

	if existing, ok := m[hash]; ok {
		existing.Count++
		existing.TotalDeltaMS += e.TotalDeltaMS
	} else {
		copyEvent := *e // Deep copy do evento
		m[hash] = &copyEvent
		*order = append(*order, hash)
	}
}

func finalize(m map[string]*models.LogEvent, order []string) []models.LogEvent {
	result := make([]models.LogEvent, 0, len(order))
	for _, key := range order {
		ev := m[key]
		if ev.Count > 0 {
			ev.AvgDeltaMS = ev.TotalDeltaMS / float64(ev.Count)
		}
		result = append(result, *ev)
	}
	return result
}

func generateHash(text string) string {
	h := md5.New()
	h.Write([]byte(text))
	return hex.EncodeToString(h.Sum(nil))
}

func containsNoise(line string, noise []string) bool {
	for _, n := range noise {
		if strings.Contains(line, n) { return true }
	}
	return false
}