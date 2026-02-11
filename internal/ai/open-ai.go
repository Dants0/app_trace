package ai

import (
    "context"
    "fmt"
    "github.com/sashabaranov/go-openai"
    "app_trace/internal/models"
)

func GetStrategicPlan(events []models.LogEvent, apiKey string) (string, error) {
    client := openai.NewClient(apiKey)
    
    // Filtramos apenas o que é relevante para economizar tokens e focar no erro
    var filteredEvents []models.LogEvent
    for _, e := range events {
        if e.Severity != "INFO" || e.Count > 10 { 
            filteredEvents = append(filteredEvents, e)
        }
    }

    prompt := fmt.Sprintf(`
        Atue como um Arquiteto de Software Especialista em PowerBuilder e Troubleshooting de Aplicações Legacy.
				Contexto: Estou enviando um arquivo de trace gerado por uma aplicação PowerBuilder (via DB Trace/ODBC). O objetivo é identificar a Causa Raiz (Root Cause) de um bug ou gargalo que está impedindo o usuário de concluir uma operação.
        Analise o rastro estruturado abaixo e retorne em no máximo 20 linhas a possível solução:
        %v

        Sua tarefa:
Instruções de Análise Obrigatórias:

Reconstrução do Fluxo de Negócio (Reverse Engineering):

Com base na sequência das queries e procedures, descreva em alto nível o que o usuário estava tentando fazer (ex: "Tentativa de login", "Processamento de Folha", "Abertura de Janela de Cadastro").

Identifique o ponto exato onde o fluxo foi interrompido (o "crash" ou o erro).

Detecção de Padrões de Erro PowerBuilder (Anti-Patterns):

Loops em Script (N+1): Identifique se há repetições excessivas de queries pequenas (indicando um SELECT ou INSERT dentro de um loop FOR no PowerScript, em vez de uma operação em lote ou DataWindow Update).

Retrieves Gigantes: Aponte se há algum SELECT trazendo colunas desnecessárias ou milhares de linhas sem paginação (típico gargalo de DataWindow.Retrieve()).

Problemas de Concorrência: Verifique se há longos períodos entre um BEGIN TRANSACTION e o COMMIT, o que pode estar travando a aplicação (Locking).

Análise do Erro/Rollback:

Se houver um ROLLBACK, identifique a última instrução executada com sucesso.

Analise se o erro foi de dados (ex: violação de constraint, tipo de dado incorreto gerado pelo PB) ou de timeout.

Sugestões de Correção para o Desenvolvedor (Action Plan):

No PowerScript: Sugira onde o código deve ser alterado (ex: "Mover a lógica de cálculo para uma Procedure", "Usar Datastore para validação", "Revisar o evento ItemChanged").

No Banco: Se necessário, sugira índices, mas priorize a lógica da aplicação.

Formato de Saída Esperado:

Diagnóstico do Cenário: O que a aplicação estava fazendo.

A Evidência do Erro: A linha ou bloco específico do trace que prova onde o bug possivelmente está.

Solução Recomendada: Passos técnicos para o desenvolvedor PowerBuilder resolver o problema.
    `, filteredEvents)

    resp, err := client.CreateChatCompletion(
        context.Background(),
        openai.ChatCompletionRequest{
            Model: openai.GPT4o,
            Messages: []openai.ChatCompletionMessage{
                {Role: openai.ChatMessageRoleSystem, Content: "Você é um expert em debugging de sistemas legados. Especificamente PowerBuilder. O seu objetivo é identificar a Causa Raiz (Root Cause) de um bug ou gargalo, a fim de facilitar a correção do Dev."},
                {Role: openai.ChatMessageRoleUser, Content: prompt},
            },
        },
    )

    if err != nil {
        return "", err
    }
    return resp.Choices[0].Message.Content, nil
}