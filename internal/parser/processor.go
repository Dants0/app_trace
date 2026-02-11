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

var numberRegex = regexp.MustCompile(`\b\d+\b`)

func ParseAndDeduplicate(scanner *bufio.Scanner) []models.LogEvent {
	aggregated := make(map[string]*models.LogEvent)
	var order []string
	var currentRawEvent *models.LogEvent
	
	noise := []string{"Length=", "BUFFER", "Bind Columns", "COLUMNS SELECTED", "VCHAR", "CHAR"}

	for scanner.Scan() {
		line := strings.Map(func(r rune) rune {
			if r < 32 && r != '\n' && r != '\t' { return -1 }
			return r
		}, scanner.Text())
		
		line = strings.TrimSpace(line)
		if line == "" || containsNoise(line, noise) {
			continue
		}

		if strings.Contains(line, " MS /") && strings.Contains(line, " MS)") {
			if currentRawEvent != nil {
				processAggregation(aggregated, &order, currentRawEvent)
			}

			openParen := strings.Index(line, "(")
			closeParen := strings.Index(line, ")")
			lastOpenParen := strings.LastIndex(line, "(")
			
			session := "unknown"
			if openParen != -1 && closeParen > openParen {
				session = line[openParen+1 : closeParen]
			}

			action := ""
			if closeParen != -1 && lastOpenParen > closeParen {
				action = strings.Trim(line[closeParen+1:lastOpenParen], " :")
			}

			delta := 0.0
			timePart := line[lastOpenParen+1:]
			if slashIdx := strings.Index(timePart, "/"); slashIdx != -1 {
				deltaStr := strings.TrimSpace(strings.ReplaceAll(timePart[:slashIdx], "MS", ""))
				delta, _ = strconv.ParseFloat(deltaStr, 64)
			}

			currentRawEvent = &models.LogEvent{
				Session:      session,
				Action:       action,
				Count:        1,
				TotalDeltaMS: delta,
			}

		} else if currentRawEvent != nil && !strings.HasPrefix(line, "(") && !strings.HasPrefix(line, "/*") {
			currentRawEvent.Statement += " " + line
		}
	}

	if currentRawEvent != nil {
		processAggregation(aggregated, &order, currentRawEvent)
	}

	return finalize(aggregated, order)
}

func setSeverity(e *models.LogEvent) {
	e.Severity = "INFO"
	
	if strings.Contains(e.Statement, "rc 100") || strings.Contains(e.Action, "DBI_FETCHEND") {
		e.Severity = "WARNING"
		e.Details = "Possível parâmetro ou registro não encontrado no banco."
	}
	
	if strings.Contains(strings.ToUpper(e.Statement), "ERROR") || strings.Contains(e.Action, "ROLLBACK") {
		e.Severity = "CRITICAL"
		e.Details = "Falha crítica na execução ou transação abortada."
	}

	if e.AvgDeltaMS > 200 {
		e.Severity = "HIGH_LATENCY"
		e.Details = "Gargalo de performance detectado."
	}
}

func processAggregation(m map[string]*models.LogEvent, order *[]string, e *models.LogEvent) {
	norm := numberRegex.ReplaceAllString(e.Statement, "?")
	hash := generateHash(e.Action + norm)

	if existing, ok := m[hash]; ok {
		existing.Count++
		existing.TotalDeltaMS += e.TotalDeltaMS
	} else {
		copyEvent := *e
		m[hash] = &copyEvent
		*order = append(*order, hash)
	}
}

func finalize(m map[string]*models.LogEvent, order []string) []models.LogEvent {
	result := make([]models.LogEvent, 0, len(order))
	for _, key := range order {
		ev := m[key]
		ev.AvgDeltaMS = ev.TotalDeltaMS / float64(ev.Count)
		setSeverity(ev) 
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

func CountBySeverity(events []models.LogEvent, severity string) int {
	count := 0
	for _, e := range events {
		if e.Severity == severity {
			count++
		}
	}
	return count
}