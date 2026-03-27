# Trace Intelligence

O **Trace Intelligence** é uma ferramenta de engenharia forense de alta performance desenvolvida em Go. Ela foi projetada para processar logs de rastro de aplicações (especificamente traces de banco de dados e PowerBuilder), identificar gargalos de performance e utilizar Inteligência Artificial para diagnosticar a causa raiz de falhas, lentidão e erros de lógica de negócio.

## 🚀 Funcionalidades

-   **Processamento Concorrente:** Processa múltiplos ficheiros de log gigabytes em paralelo usando Goroutines.
-   **Template Mining & Deduplicação:** Normaliza queries SQL e scripts (substituindo números por placeholders) para agrupar execuções similares e calcular médias reais de tempo.
-   **Diagnóstico via IA (OpenAI):** Integração com LLMs para atuar como um "Especialista em PowerBuilder", identificando loops (N+1), locks de transação e erros de lógica.
-   **Métricas de Severidade:** Classificação automática de eventos críticos baseada em tempo de execução e códigos de erro.
-   **API REST:** Endpoints robustos para integração com dashboards ou ferramentas de CI/CD.

## 🛠️ Stack Tecnológica

-   **Linguagem:** Go (Golang) 1.25+
-   **Web Framework:** [Gin Gonic](https://github.com/gin-gonic/gin)
-   **Configuração:** [Godotenv](https://github.com/joho/godotenv)
-   **IA:** Integração via REST com OpenAI API (GPT-4/GPT-3.5)

## 📋 Pré-requisitos

-   Go instalado na máquina.
-   Uma chave de API da OpenAI (para as funcionalidades de IA).

## ⚙️ Configuração e Instalação

1.  **Clone o repositório:**
    ```bash
    git clone [https://github.com/dants0/app-trace.git](https://github.com/dants0/app-trace.git)
    cd app-trace
    ```

2.  **Instale as dependências:**
    ```bash
    go get
    go mod tidy
    ```

3.  **Configure as Variáveis de Ambiente:**
    Crie um arquivo `.env` na raiz do projeto e adicione suas credenciais:

    ```env
    PORT=:8000
    OPENAI_API_KEY=sk-sua-chave-api-aqui
    GEMINI_API_KEY=sua_chave_gemini
    AI_PROVIDER=gemini # ou 'openai'
    ```

4.  **Execute a aplicação:**
    ```bash
    go run main.go
    ```
    O servidor iniciará na porta definida (padrão `:8000`).

## 🔌 Documentação da API

A aplicação expõe três endpoints principais via `POST`. Todos esperam o envio de arquivos via `multipart/form-data` no campo `files`.

### 1. Processamento Básico (`/process-batch`)
Processa os logs, limpa ruídos e retorna os dados estruturados e deduplicados. Ideal para ingestão de dados brutos.

-   **URL:** `/process-batch`
-   **Método:** `POST`
-   **Retorno Exemplo:**
    ```json
    [
      {
        "filename": "trace_01.log",
        "event_count": 1500,
        "data": [
          {
            "session": "04547C10",
            "action": "PREPARE",
            "statement": "SELECT * FROM users WHERE id = ?",
            "count": 50,
            "avg_delta_ms": 12.5
          }
        ]
      }
    ]
    ```

### 2. Análise Estratégica com IA (`/analyze-ai`)
Envia os dados processados para a IA, que retorna um plano de ação focado em corrigir a lógica da aplicação (ex: PowerBuilder DataWindows, Loops).

-   **URL:** `/analyze-ai`
-   **Método:** `POST`
-   **Retorno Exemplo:**
    ```json
    [
      {
        "filename": "app_trace.log",
        "event_count": 850,
        "strategic_plan": "A análise detectou um loop 'N+1' na sessão X. O usuário tentava processar a folha, mas o script realiza um SELECT unitário para cada funcionário...",
        "structured_data": [...]
      }
    ]
    ```

### 3. Análise Completa de Trace (`/analyze-trace`)
O endpoint mais completo. Retorna um sumário executivo com contagem de eventos críticos, além da análise da IA e dos dados estruturados.

-   **URL:** `/analyze-trace`
-   **Método:** `POST`
-   **Retorno Exemplo:**
    ```json
    [
      {
        "filename": "debug_full.log",
        "summary": {
          "total_events": 5000,
          "critical_count": 12
        },
        "strategic_analysis": "O Rollback foi causado por um deadlock na tabela 'cfg'...",
        "structured_data": [...]
      }
    ]
    ```

## 🧠 Como funciona o Parser Interno

O processador (`internal/parser`) executa as seguintes etapas:

1.  **Sanitização:** Remove caracteres inválidos e linhas de "ruído" (ex: buffers internos, bind columns).
2.  **Identificação de Padrões:** Detecta assinaturas de tempo `(0.000 MS / 10.000 MS)` para separar comandos.
3.  **Template Mining:** Usa Regex para generalizar queries.
    * *Entrada:* `SELECT * FROM t WHERE id = 105`
    * *Processado:* `SELECT * FROM t WHERE id = ?`
    * Isso permite agrupar milhares de execuções diferentes em uma única métrica estatística.
4.  **Hashing:** Gera um ID único (MD5) para cada padrão de query para deduplicação rápida.
