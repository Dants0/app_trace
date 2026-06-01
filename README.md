Aqui está a versão atualizada do seu `README.md`, refletindo as últimas melhorias de arquitetura, a nova porta padrão (8070), o suporte expandido a múltiplos modelos de IA e a integração com o Docker.

---

# Trace Intelligence Pro

O **Trace Intelligence Pro** é uma ferramenta de engenharia forense de alta performance desenvolvida em Go. Ela foi projetada para processar logs de rastro de aplicações (especificamente traces de banco de dados e PowerBuilder), identificar gargalos de performance e utilizar Inteligência Artificial para diagnosticar a causa raiz de falhas, lentidão e erros de lógica de negócio.

## 🚀 Funcionalidades

- **Processamento Concorrente:** Processa múltiplos arquivos de log (gigabytes) em paralelo usando Goroutines.
- **Template Mining & Deduplicação:** Normaliza queries SQL e scripts (substituindo números por placeholders) para agrupar execuções similares e calcular médias reais de tempo.
- **Diagnóstico Multi-IA:** Integração flexível com múltiplos LLMs para atuar como um "Especialista em PowerBuilder", identificando loops (N+1), locks de transação e erros de lógica.
- **Métricas de Severidade:** Classificação automática de eventos críticos baseada no tempo de execução e na quantidade de linhas afetadas.
- **Frontend Integrado:** A interface de usuário (HTML/CSS/JS) é servida de forma nativa e otimizada pelo próprio binário Go.
- **Containerização:** Aplicação totalmente dockerizada via Multi-stage Build, garantindo deploys leves e execução isolada.

## 🛠️ Stack Tecnológica

- **Backend:** Go (Golang) 1.25+
- **Web Framework:** [Gin Gonic](https://github.com/gin-gonic/gin)
- **Engines de IA Suportados:** OpenAI (GPT-4o), Google Gemini, Anthropic (Claude), Groq (Llama) e Azure (Copilot).
- **Infraestrutura:** Docker & Docker Compose.

## 📋 Pré-requisitos

- **Docker e Docker Compose** (Recomendado para execução imediata).
- **Go 1.25+** instalado na máquina (Apenas se for rodar localmente sem Docker).
- Chaves de API das IAs que deseja utilizar.

## ⚙️ Configuração e Instalação

1.  **Clone o repositório:**
    ```bash
    git clone https://github.com/dants0/app-trace.git
    cd app-trace
    ```

2.  **Configure as Variáveis de Ambiente:**
    Crie um arquivo `.env` na raiz do projeto e adicione suas credenciais (as chaves também podem ser configuradas diretamente na interface web):
    ```env
    OPENAI_API_KEY=sua_chave_openai
    GEMINI_API_KEY=sua_chave_gemini
    ANTHROPIC_API_KEY=sua_chave_anthropic
    GROQ_API_KEY=sua_chave_groq
    AZURE_OPENAI_ENDPOINT=https://sua-empresa.openai.azure.com/
    AZURE_OPENAI_KEY=sua_chave_azure
    PORT=:8070
    ```

### 🐳 Execução via Docker (Recomendado)

A forma mais rápida e limpa de rodar a aplicação em qualquer ambiente:

```bash
docker-compose up -d --build
```
Acesse a aplicação no seu navegador: **`http://localhost:8070`**

### 💻 Execução Local (Modo Dev)

Caso prefira rodar usando a instalação local do Go:

```bash
go mod tidy
go run main.go
```
Acesse: **`http://localhost:8070`**

## 🔌 Documentação da API

Além de servir a interface visual na raiz (`/`), a aplicação expõe endpoints RESTful via `POST`. Todos esperam o envio de arquivos via `multipart/form-data` no campo `files` e aceitam as chaves de API pelo payload do form.

### 1. Processamento Básico (`/process-batch`)
Processa os logs, limpa ruídos e retorna os dados estruturados e deduplicados. Ideal para ingestão de dados brutos.

- **Método:** `POST`
- **Retorno Exemplo:**
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
          "avg_delta_ms": 12.5,
          "severity": "INFO"
        }
      ]
    }
  ]
  ```

### 2. Análise Estratégica com IA (`/analyze-ai`)
Envia os dados processados para a IA selecionada, retornando um plano de ação focado em corrigir a lógica da aplicação (ex: PowerBuilder DataWindows, Loops).

- **Método:** `POST`
- **Retorno Exemplo:**
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
O endpoint mais completo. Executa todo o pipeline de processamento e diagnóstico, devolvendo a estrutura final pronta para renderização.

- **Método:** `POST`
- **Retorno Exemplo:**
  ```json
  [
    {
      "filename": "debug_full.log",
      "event_count": 5000,
      "strategic_analysis": "**Diagnóstico do Cenário:** O Rollback foi causado por um...",
      "structured_data": [...]
    }
  ]
  ```

### 4. Health da API (`/health`)
- **Método:** `GET`
- **Retorno Exemplo:**
  ```json
  {
    "status": "ok",
    "timestamp": 1780318569,
    "version": "3.1.0"
  }
  ```



## 🧠 Como funciona o Parser Interno

O processador (`internal/parser/processor.go`) executa as seguintes etapas para garantir precisão e velocidade:

1.  **Sanitização:** Remove caracteres inválidos e ignora linhas de "ruído" (ex: buffers internos, bind columns).
2.  **Agrupamento Temporal:** Detecta quebras lógicas e timestamps para criar "blocos" de execução (`TraceGroup`).
3.  **Identificação de Padrões e Deltas:** Detecta assinaturas de tempo `(0.000 MS / 10.000 MS)` para extrair a latência exata e ação executada.
4.  **Template Mining:** Usa Regex para generalizar queries.
    * *Entrada:* `SELECT * FROM t WHERE id = 105`
    * *Processado:* `SELECT * FROM t WHERE id = ?`
5.  **Hashing e Deduplicação:** Gera um ID único (MD5) para cada padrão de query. Isso agrupa milhares de execuções idênticas em uma única métrica estatística (contagem e tempo médio).
6.  **Atribuição de Severidade:** Classifica automaticamente os eventos como `CRITICAL`, `WARNING` ou `HIGH_LATENCY` baseando-se em timeouts, erros ou zero linhas afetadas (`RowsAffected`).