package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"app_trace/internal/models"

	"github.com/google/generative-ai-go/genai"
	googleOption "google.golang.org/api/option" // Alias para a lib do Google

	"github.com/sashabaranov/go-openai"
)

// GetStrategicPlan atua como um Router (Factory) para direcionar a chamada para a IA correta
func GetStrategicPlan(groups []models.TraceGroup, bugDescription string, modelAi string) (string, error) {
	if strings.Contains(modelAi, "gemini") {
		return callGemini(groups, bugDescription, modelAi)
	}
	if strings.Contains(modelAi, "claude") {
		return callAnthropic(groups, bugDescription, modelAi)
	}
	if strings.Contains(modelAi, "llama") {
		return callLlamaViaGroq(groups, bugDescription, modelAi)
	}
	if strings.Contains(modelAi, "copilot") {
		return callCopilot(groups, bugDescription, modelAi)
	}
	return callOpenAI(groups, bugDescription, modelAi)
}

// -----------------------------------------------------------------------------
// FUNÇÕES AUXILIARES
// -----------------------------------------------------------------------------

func filterEvents(groups []models.TraceGroup) []models.TraceGroup {
	var filteredGroups []models.TraceGroup
	for _, g := range groups {
		var filtered []models.LogEvent
		for _, e := range g.Events {
			actionUpper := strings.ToUpper(e.Action)
			if strings.Contains(actionUpper, "GET EXTENDED ATTRIBUTES") ||
				strings.Contains(actionUpper, "LOGIN") ||
				strings.Contains(actionUpper, "UNIQUE KEY CHECK") ||
				strings.Contains(actionUpper, "DESCRIBE") ||
				strings.Contains(actionUpper, "BLOB READ") ||
				strings.Contains(actionUpper, "DBPARM=CONNECTSTRING") {
				continue
			}
			filtered = append(filtered, e)
		}
		if len(filtered) > 0 {
			filteredGroups = append(filteredGroups, models.TraceGroup{
				Timestamp: g.Timestamp,
				Events:    filtered,
			})
		}
	}
	return filteredGroups
}

func buildPrompt(filteredEvents []models.TraceGroup, bugDesc string) string {
	return fmt.Sprintf(`
Você é um Arquiteto de Software Sênior especialista em PowerBuilder e SQL Tuning.
Sua missão é fazer o troubleshooting de uma aplicação legada analisando um DB Trace.

Contexto do Erro relatado pelo usuário/QA: "%s"

Abaixo está o log de trace filtrado da sessão onde o erro ocorreu:
%v

Instruções rigorosas para sua análise:
1. FOCO NO NEGÓCIO: Ignore queries de infraestrutura (login, controle de sessão). Vá direto para as transações próximas ao momento do erro relatado.
2. ANÁLISE DE SQL: Procure ativamente por falhas na cláusula WHERE das consultas. O erro é causado por um SELECT que traz dados a mais ou um UPDATE sem restrição adequada?
3. SINTOMAS POWERBUILDER: Verifique se há sinais de N+1 (loops no PowerScript fazendo queries individuais em vez de DataWindows) ou DataWindows puxando milhares de linhas sem paginação.

Estruture sua resposta EXATAMENTE neste formato (seja técnico, direto e não use mais que 4 parágrafos):

**Diagnóstico do Cenário:** Explique a lógica de banco de dados que a aplicação tentou executar relacionada ao erro.
**Evidência do Erro:** Mostre a query específica do trace que está causando o problema e explique o que está faltando nela ou o gargalo.
**Solução Recomendada:** Dê o plano de ação técnico para o dev arrumar no PowerScript ou no banco de dados.
`, bugDesc, filteredEvents)
}

func callGemini(groups []models.TraceGroup, bugDesc string, modelAi string) (string, error) {
	ctx := context.Background()
	apiKey := os.Getenv("GEMINI_API_KEY")
	client, err := genai.NewClient(ctx, googleOption.WithAPIKey(apiKey))
	if err != nil {
		return "", err
	}
	defer client.Close()

	model := client.GenerativeModel(modelAi)

	prompt := buildPrompt(filterEvents(groups), bugDesc)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", err
	}

	if len(resp.Candidates) > 0 {
		return fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0]), nil
	}
	return "Nenhuma análise gerada", nil
}

func callOpenAI(groups []models.TraceGroup, bugDesc string, modelAi string) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	client := openai.NewClient(apiKey)

	filteredEvents := filterEvents(groups)
	prompt := buildPrompt(filteredEvents, bugDesc)

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: modelAi,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: "Você é um expert em debugging de sistemas legados PowerBuilder."},
				{Role: openai.ChatMessageRoleUser, Content: prompt},
			},
		},
	)

	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}

func callAnthropic(groups []models.TraceGroup, bugDesc string, modelAi string) (string, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("ANTHROPIC_API_KEY não configurada")
	}

	filteredEvents := filterEvents(groups)
	prompt := buildPrompt(filteredEvents, bugDesc)

	// Estruturas locais limpas para montar o JSON da requisição
	type Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	type RequestBody struct {
		Model     string    `json:"model"`
		System    string    `json:"system"`
		MaxTokens int       `json:"max_tokens"`
		Messages  []Message `json:"messages"`
	}

	reqBody := RequestBody{
		Model:     modelAi,
		System:    "Você é um expert em debugging de sistemas legados PowerBuilder.",
		MaxTokens: 2000,
		Messages: []Message{
			{Role: "user", Content: prompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("erro da API Anthropic (Status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var responseStruct struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.Unmarshal(bodyBytes, &responseStruct); err != nil {
		return "", err
	}

	if len(responseStruct.Content) > 0 {
		return responseStruct.Content[0].Text, nil
	}

	return "Nenhuma análise gerada", nil
}

func callLlamaViaGroq(groups []models.TraceGroup, bugDesc string, modelAi string) (string, error) {
	apiKey := os.Getenv("GROQ_API_KEY")

	config := openai.DefaultConfig(apiKey)
	config.BaseURL = "https://api.groq.com/openai/v1"
	client := openai.NewClientWithConfig(config)

	filteredEvents := filterEvents(groups)
	prompt := buildPrompt(filteredEvents, bugDesc)

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: modelAi,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: "Você é um expert em debugging de sistemas legados PowerBuilder."},
				{Role: openai.ChatMessageRoleUser, Content: prompt},
			},
		},
	)

	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}

func callCopilot(groups []models.TraceGroup, bugDesc string, modelAi string) (string, error) {
	apiKey := os.Getenv("AZURE_OPENAI_API_KEY")
	endpoint := os.Getenv("AZURE_OPENAI_ENDPOINT")

	if apiKey == "" || endpoint == "" {
		return "", fmt.Errorf("credenciais do Azure OpenAI (Copilot) não configuradas no .env")
	}

	config := openai.DefaultAzureConfig(apiKey, endpoint)
	client := openai.NewClientWithConfig(config)

	filteredEvents := filterEvents(groups)
	prompt := buildPrompt(filteredEvents, bugDesc)

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: modelAi,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: "Você é um expert em debugging de sistemas legados PowerBuilder."},
				{Role: openai.ChatMessageRoleUser, Content: prompt},
			},
		},
	)

	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}
