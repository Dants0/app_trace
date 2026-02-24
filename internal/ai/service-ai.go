package ai

import (
	"context"
	"fmt"
	"os"
	"strings"

	"app_trace/internal/models"

	"github.com/google/generative-ai-go/genai"
	"github.com/sashabaranov/go-openai"
	"google.golang.org/api/option"
)

func GetStrategicPlan(events []models.LogEvent, bugDescription string, modelAi string) (string, error) {
	if strings.Contains(modelAi, "gemini") {
		return callGemini(events, bugDescription, modelAi)
	}
	return callOpenAI(events, bugDescription, modelAi)
}

func callGemini(events []models.LogEvent, bugDesc string, modelAi string) (string, error) {
	ctx := context.Background()
	apiKey := ""
	apiKey = os.Getenv("GEMINI_API_KEY")
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))

	if err != nil {
		return "", err
	}
	defer client.Close()

	model := client.GenerativeModel(modelAi)

	prompt := fmt.Sprintf(`
		Aja como Arquiteto de Software especialista em PowerBuilder e SQL Server.
		CONTEXTO DO ERRO: %s
		DADOS DO RASTRO: %v
		TAREFA: Analise o rastro, identifique rc 100 em tabelas 'cfg' ou 'ini', falhas de schema e proponha correção técnica.
	`, bugDesc, events)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", err
	}

	if len(resp.Candidates) > 0 {
		return fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0]), nil
	}
	return "Nenhuma análise gerada", nil
}

func callOpenAI(events []models.LogEvent, bugDesc string, modelAi string) (string, error) {
	apiKey := ""
	apiKey = os.Getenv("OPENAI_API_KEY")
	client := openai.NewClient(apiKey)

	var filteredEvents []models.LogEvent
	for _, e := range events {
		actionUpper := strings.ToUpper(e.Action)

		if strings.Contains(actionUpper, "GET EXTENDED ATTRIBUTES") ||
		 	strings.Contains(actionUpper, "LOGIN") ||
			strings.Contains(actionUpper, "UNIQUE KEY CHECK") ||
			strings.Contains(actionUpper, "DESCRIBE") ||
			strings.Contains(actionUpper, "BLOB READ") ||
			strings.Contains(actionUpper, "DBPARM=CONNECTSTRING") {
			continue
		}

		filteredEvents = append(filteredEvents, e)
	}

	prompt := fmt.Sprintf(`
Você é um Arquiteto de Software Sênior especialista em PowerBuilder e SQL Tuning.
Sua missão é fazer o troubleshooting de uma aplicação legada analisando um DB Trace.

Contexto do Erro relatado pelo usuário/QA: "%s"

Abaixo está o log de trace filtrado da sessão onde o erro ocorreu:
%%v

Instruções rigorosas para sua análise:
1. FOCO NO NEGÓCIO: Ignore queries de infraestrutura (login, controle de sessão). Vá direto para as transações próximas ao momento do erro relatado.
2. ANÁLISE DE SQL (Missing Predicates): Procure ativamente por falhas na cláusula WHERE das consultas. O erro é causado por um SELECT que está trazendo dados a mais (falta de filtro de tipo/classe) ou um UPDATE sem restrição adequada?
3. SINTOMAS POWERBUILDER: Verifique se há sinais de N+1 (loops no PowerScript fazendo queries individuais em vez de DataWindows) ou DataWindows puxando milhares de linhas sem paginação.

Estruture sua resposta EXATAMENTE neste formato (seja técnico, direto e não use mais que 4 parágrafos):

**Diagnóstico do Cenário:** Explique a lógica de banco de dados que a aplicação tentou executar relacionada ao erro.
**Evidência do Erro:** Mostre a query específica do trace que está causando o problema e explique o que está faltando nela (ex: um filtro WHERE) ou o gargalo.
**Solução Recomendada:** Dê o plano de ação técnico para o dev arrumar no PowerScript ou no banco de dados.
`, bugDesc)

	finalPrompt := fmt.Sprintf(prompt, filteredEvents)

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: modelAi,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: "Você é um expert em debugging de sistemas legados. Especificamente PowerBuilder. O seu objetivo é identificar a Causa Raiz (Root Cause) de um bug ou gargalo, a fim de facilitar a correção do Dev."},
				{Role: openai.ChatMessageRoleUser, Content: finalPrompt},
			},
		},
	)

	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}
