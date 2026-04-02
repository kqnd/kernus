# Kernus — Guidelines para Recriação do Projeto

> Este documento é um guia estruturado para recriar o Kernus do zero.
> Siga as etapas em ordem. Cada etapa é independente e buildável.
> Ao final de cada etapa, o projeto deve compilar e funcionar.
> e nao coloque comentarios no codigo

---

## Visão Geral do Projeto

**Nome:** Kernus (CLI: `kernus`)
**Linguagem:** Go 1.23+
**Tipo:** CLI/TUI — Terminal User Interface para monitoramento de infraestrutura

**Objetivo:** Criar uma ferramenta de terminal que permita ao usuário:
1. Autenticar com um servidor de monitoramento remoto (login, perfil, logout)
2. Visualizar e gerenciar containers Docker locais em tempo real
3. Monitorar métricas de máquinas remotas (CPU, RAM, disco, processos)
4. Controlar containers (start/stop/pause/restart/remove) via atalhos de teclado
5. Ver logs, estatísticas detalhadas e configuração de rede de cada container

**Diferenciais em relação a ferramentas como `lazydocker` ou `ctop`:**
- Sistema de autenticação integrado — login, perfil e logout tudo pelo terminal
- Monitoramento de múltiplas máquinas remotas num único painel
- Agrupamento hierárquico de containers por prefixo de nome (expandir/colapsar grupos)
- Parsing avançado de logs com detecção de nível (ERROR/WARN/INFO/DEBUG) e cores
- Arquitetura extensível para suportar backends remotos

---

## Stack Técnica

```
github.com/spf13/cobra      → CLI framework
github.com/rivo/tview       → TUI framework
github.com/gdamore/tcell/v2 → Terminal cell layer
github.com/docker/docker    → Docker SDK oficial
```

**Estrutura de módulo:**
```
module github.com/<seu-usuario>/kernus

go 1.23.0
```

---

## Estrutura de Diretórios Final

```
kernus/
├── main.go
├── go.mod
├── go.sum
├── .gitignore
├── Makefile
├── cmd/
│   ├── root.go         → Comando raiz, carrega config
│   ├── config.go       → Subcomando de configuração
│   ├── login.go        → Login interativo no servidor
│   ├── logout.go       → Encerra sessão
│   ├── profile.go      → Exibe perfil do usuário logado
│   ├── see.go          → Lança TUI de monitoramento
│   └── send.go         → Envia métricas desta máquina
├── internal/
│   ├── auth/
│   │   ├── client.go   → Comunicação de auth com o servidor
│   │   └── session.go  → Gerenciamento de sessão (token, cache)
│   ├── config/
│   │   └── config.go   → Leitura/escrita de config segura
│   ├── docker/
│   │   ├── client.go   → Wrapper do Docker SDK
│   │   └── mock.go     → Mock client para testes
│   ├── metrics/
│   │   └── collector.go → Coleta métricas locais da máquina
│   ├── models/
│   │   ├── container.go
│   │   ├── machine.go
│   │   └── user.go     → Modelo do usuário autenticado
│   └── tui/
│       ├── app.go
│       ├── login_app.go → TUI de login (tela separada)
│       └── components/
│           ├── header.go
│           ├── containers.go
│           ├── machines.go
│           ├── details.go
│           ├── statusbar.go   → Barra inferior com atalhos
│           ├── modal.go       → Modal de confirmação
│           ├── login_form.go  → Formulário de login TUI
│           ├── profile_panel.go → Painel de perfil do usuário
│           └── details/
│               ├── formatter.go
│               ├── visualizer.go
│               ├── table_builder.go
│               ├── overview.go
│               ├── stats.go
│               ├── network.go
│               ├── storage.go
│               └── logs.go
```

---

## Etapa 1 — Fundação do Projeto

**Objetivo:** Projeto Go compilável com CLI funcional e gerenciamento de configuração seguro.

### 1.1 Setup Inicial

- Criar `go.mod` com o nome do módulo
- Criar `.gitignore` incluindo: `config.json`, `/kernus` (binário), `*.exe`, `.env`
- Criar `Makefile` com targets: `build`, `run`, `test`, `lint`, `clean`
- Criar `main.go` com apenas `cmd.Execute()`

### 1.2 Sistema de Configuração

Criar `internal/config/config.go` com as seguintes regras:

**Campos da configuração:**
```go
type Config struct {
    Server   string // URL WebSocket: ws:// ou wss://
    Username string
    Password string // nunca logar este campo
    Database string
    Token    string // nunca logar este campo
}
```

**Regras de segurança e comportamento:**
- O arquivo de config deve ser salvo em `~/.config/kernus/config.json` (via `os.UserConfigDir()`), NUNCA no diretório do projeto
- Permissões do arquivo: `0600` (somente leitura/escrita pelo owner)
- Criar o diretório se não existir, com permissão `0700`
- Tratar TODOS os erros explicitamente — nunca usar `_` para ignorar erros
- Usar `io.ReadAll` (não `ioutil.ReadAll` — está deprecado desde Go 1.16)
- Fazer `json.Unmarshal` e retornar erro se falhar
- Expor função `Validate() error` que retorna erro descritivo se campos obrigatórios estiverem vazios

### 1.3 Comandos CLI (Cobra)

**`kern`** — Comando raiz
- Carrega config no `PersistentPreRunE` (não no `init()`)
- Mostra help formatado com exemplos

**`kern config`** — Configuração
- Flags: `--server`, `--username`, `--password`
- Todas obrigatórias, validar antes de salvar
- Mostrar caminho do arquivo após salvar com sucesso

**`kern login`** — Login no servidor
- Se executado sem flags, lança a **TUI de login** (formulário interativo no terminal)
- Flags opcionais para uso não-interativo: `--username`, `--password`
- Após login com sucesso: salvar token de sessão em `~/.config/kernus/session.json` com permissão `0600`
- Exibir mensagem: `✓ Logged in as [username] — session expires in 24h`
- Se já há sessão ativa, perguntar se deseja fazer login de novo

**`kern logout`** — Encerra sessão
- Invalidar token no servidor (chamar endpoint de logout)
- Deletar `~/.config/kernus/session.json`
- Exibir: `✓ Logged out successfully`

**`kern profile`** — Exibe perfil do usuário logado
- Requer sessão ativa (retornar erro claro se não logado)
- Buscar dados do servidor e exibir no terminal:
  ```
  ┌─────────────────────────────┐
  │  Perfil do Usuário          │
  │  Username : john_doe        │
  │  Email    : john@acme.com   │
  │  Role     : admin           │
  │  Groups   : backend, web    │
  │  Last Login: 2026-03-23     │
  └─────────────────────────────┘
  ```
- Flag `--json` para output em JSON (útil para scripts)

**`kern see`** — Lança TUI
- **Verificar sessão antes de iniciar:** se não há sessão ativa, redirecionar automaticamente para a TUI de login
- Flag: `--group` (filtra máquinas por grupo)
- Flag: `--docker-host` (URL do Docker daemon, padrão: socket local)
- Flag: `--refresh` (intervalo em segundos, padrão: 3)
- Flag: `--machines` (mostra painel de máquinas remotas em vez de containers locais)

**`kern send`** — Envia métricas (implementar na Etapa 6)
- Flag: `--name` (nome da máquina, obrigatório)
- Flag: `--group` (grupo da máquina, padrão: "default")
- Flag: `--interval` (intervalo de envio em segundos, padrão: 5)

---

## Etapa 2 — Models de Dados

**Objetivo:** Definir todas as estruturas de dados com seus métodos auxiliares.

### 2.1 `internal/models/container.go`

**Structs principais:**
- `Container`: ID, Name, Image, Status, State, Created, Started, Finished, Ports, Mounts, Networks, Labels, Command, Stats, Health, RestartPolicy, ExitCode, Logs
- `ContainerStats`: CPU, Memory, Network, BlockIO, PIDs, Timestamp
- `ContainerCPU`: Usage (%), System (%), Cores, Throttling
- `ContainerMemory`: Usage, Limit, Cache, RSS, Swap, MaxUsage
- `ContainerNetwork`: RxBytes, RxPackets, RxErrors, RxDropped, TxBytes, TxPackets, TxErrors, TxDropped
- `ContainerBlockIO`: ReadBytes, WriteBytes, ReadOps, WriteOps
- `ContainerHealth`: Status, FailingStreak, Log
- `Port`, `Mount`, `Network`, `RestartPolicy`

**Tipos e constantes:**
- `ContainerStatus` string type com constantes: `running`, `exited`, `paused`, `stopped`, `created`, `restarting`, `removing`, `dead`
- `HealthStatus` com constantes: `healthy`, `unhealthy`, `starting`, `none`
- Cada tipo deve ter métodos `Color() string` e `Icon() string`

**Métodos do Container:**
- `ShortID() string` — primeiros 12 chars do ID
- `ShortName() string` — nome sem prefixo `/`
- `Age() time.Duration`, `FormatAge() string`
- `Uptime() time.Duration`, `FormatUptime() string`
- `MainPort() string`, `AllPorts() string`, `ShortPort() string`
- `ImageName() string`, `ImageTag() string`
- `IsHealthy() bool`
- `GetCPUUsage() float64`, `GetMemoryUsage() int64`, `GetMemoryLimit() int64`
- `GetRecentLogs(maxLines int) []string`

**Métodos dos stats:**
- `ContainerMemory.Percentage() float64`
- `ContainerMemory.String() string` — "50.0MB / 512.0MB (9.8%)"
- `ContainerNetwork.String() string` — "↓ 100.0MB ↑ 200.0MB"
- `ContainerBlockIO.String() string`
- `ContainerCPU.ThrottleString() string`

**Função `MockContainers() []Container`** — retorna ao menos 5 containers fictícios com dados realistas para desenvolvimento/debug.

### 2.2 `internal/models/user.go`

**Structs:**
- `User`: ID, Username, Email, Role, Groups, CreatedAt, LastLogin
- `Session`: Token, Username, UserID, ExpiresAt, CreatedAt

**Tipos:**
- `Role` string type com constantes: `admin`, `operator`, `viewer`
- `Role.CanManageContainers() bool` — true para admin e operator
- `Role.CanRemove() bool` — true apenas para admin

**Métodos da Session:**
- `IsExpired() bool` — compara `ExpiresAt` com `time.Now()`
- `TimeUntilExpiry() time.Duration`
- `FormatExpiry() string` — "expires in 23h45m"

**Função `MockUser() User`** — dados fictícios para desenvolvimento.

### 2.3 `internal/models/machine.go`

**Structs:**
- `Machine`: ID, Name, Status, CPUUsage, MemoryUsage, DiskUsage, IP, LastSeen, Uptime, Processes, Group
- `Memory`: Used, Total → métodos `Percentage() float64`, `String() string`
- `Disk`: Used, Total → métodos `Percentage() float64`, `String() string`
- `Duration`: Seconds int64 → método `String() string` (human-readable: "2 days, 3 hours")
- `Process`: Address, Port, Name
- `Group`: Name, MachineCount

**`Status` type** com: `online`, `offline`, `error` + método `Color() string`

**Função `MockMachines() []Machine`** — retorna ao menos 8 máquinas em grupos diferentes (frontend, database, cache, backend, monitoring).

---

## Etapa 3 — Autenticação

**Objetivo:** Sistema completo de login, sessão e perfil integrado ao CLI e à TUI.

### 3.1 Session Manager (`internal/auth/session.go`)

**Responsabilidades:**
- Ler/escrever `~/.config/kernus/session.json` com permissão `0600`
- Struct salva em disco:
  ```go
  type StoredSession struct {
      Token     string    `json:"token"`
      Username  string    `json:"username"`
      UserID    string    `json:"user_id"`
      ExpiresAt time.Time `json:"expires_at"`
      Server    string    `json:"server"` // URL do servidor desta sessão
  }
  ```
- Funções:
  - `SaveSession(s *StoredSession) error`
  - `LoadSession() (*StoredSession, error)` — retorna `ErrNoSession` se não existe
  - `DeleteSession() error`
  - `IsSessionValid() bool` — carrega e verifica expiração

**Nunca logar o campo `Token`.**

### 3.2 Auth Client (`internal/auth/client.go`)

Interface:
```go
type AuthClient interface {
    Login(ctx context.Context, username, password string) (*models.Session, error)
    Logout(ctx context.Context, token string) error
    GetProfile(ctx context.Context, token string) (*models.User, error)
    ValidateToken(ctx context.Context, token string) (bool, error)
}
```

**Implementação HTTP/WebSocket:**
- `Login`: POST para `{server}/auth/login` com JSON body `{username, password}`
  - Resposta esperada: `{token, expires_at, user_id, username}`
  - Em caso de credenciais inválidas, retornar `ErrInvalidCredentials` (nunca expor detalhes internos)
  - Timeout de 10 segundos
- `Logout`: POST para `{server}/auth/logout` com header `Authorization: Bearer {token}`
- `GetProfile`: GET para `{server}/auth/profile` com header `Authorization: Bearer {token}`
- `ValidateToken`: GET para `{server}/auth/validate` — usado na inicialização do `kern see`

**Erros tipados:**
```go
var (
    ErrInvalidCredentials = errors.New("invalid username or password")
    ErrSessionExpired     = errors.New("session has expired, please login again")
    ErrNoSession          = errors.New("not logged in — run 'kern login'")
    ErrServerUnreachable  = errors.New("cannot reach server")
)
```

### 3.3 TUI de Login (`internal/tui/login_app.go`)

Tela de login que ocupa o terminal inteiro antes de entrar no monitoramento:

```
┌─────────────────────────────────────────────────────────┐
│                                                         │
│                    K E R N U S                          │
│              Infrastructure Monitor                     │
│                                                         │
│  ┌───────────────────────────────────────────────────┐  │
│  │                    Login                         │  │
│  │                                                  │  │
│  │  Server  : ws://monitoring.acme.com              │  │
│  │                                                  │  │
│  │  Username: [_________________________________]   │  │
│  │  Password: [*********************************]   │  │
│  │                                                  │  │
│  │              [ Entrar ]  [ Sair ]                │  │
│  │                                                  │  │
│  │  [red]✗ Invalid credentials. Try again.[white]   │  │
│  └───────────────────────────────────────────────────┘  │
│                                                         │
│        [Tab] Navegar  [Enter] Confirmar  [Esc] Sair     │
└─────────────────────────────────────────────────────────┘
```

**Comportamento:**
- Usar `tview.Form` com campos Username e Password (Password com `SetMaskCharacter('*')`)
- Exibir servidor configurado acima do formulário (somente leitura)
- Campo de erro abaixo dos botões — começa vazio, exibe em vermelho se login falhar
- Ao pressionar Enter no campo Password ou clicar em Entrar: chamar `AuthClient.Login`
- Durante requisição: desabilitar botão e exibir `Conectando...`
- Após login: fechar esta tela e abrir a TUI de monitoramento
- Após 3 tentativas falhas: adicionar delay de 3 segundos antes de permitir nova tentativa

**Componente reutilizável** `components/login_form.go`:
- Pode ser embutido em qualquer tela como modal ou tela cheia
- Callback `OnSuccess func(session *models.Session)`
- Callback `OnCancel func()`

### 3.4 Painel de Perfil na TUI (`components/profile_panel.go`)

Acessível dentro do `kern see` pelo atalho `i` (informações do usuário):

```
┌──────────────────────────────────────────┐
│  Perfil do Usuário                       │
│                                          │
│  Username : john_doe                     │
│  Email    : john@acme.com                │
│  Role     : [green]admin[white]          │
│  Groups   : backend, web, monitoring     │
│                                          │
│  Sessão atual:                           │
│  Expira em  : 23h 45m                    │
│  Servidor   : ws://monitoring.acme.com   │
│                                          │
│  [ Fechar ]          [ Logout ]          │
└──────────────────────────────────────────┘
```

- Implementar como modal (`tview.Modal` + `tview.TextView`)
- Botão Logout: chamar `AuthClient.Logout`, deletar sessão, fechar TUI e sair
- Role com cor: admin=verde, operator=amarelo, viewer=cinza

### 3.5 Integração com `kern see`

No início do `app.Run()`:
1. Carregar sessão com `LoadSession()`
2. Se `ErrNoSession` ou sessão expirada: exibir TUI de login antes de continuar
3. Se sessão válida mas expira em menos de 1 hora: exibir aviso no header
4. Ao longo da sessão, se uma requisição retornar `401 Unauthorized`: exibir modal de re-login sem fechar a TUI

**Header com informações de usuário:**
```
 Server: ws://... | Usuário: john_doe [admin] | Hora: 15:04:05 | ● Connected
```

---

## Etapa 4 — Docker Client

**Objetivo:** Wrapper completo do Docker SDK com interface testável.

### 3.1 Interface

Criar uma interface `DockerClient` em `internal/docker/client.go`:

```go
type DockerClient interface {
    ListContainers(ctx context.Context, onlyRunning bool) ([]models.Container, error)
    StartContainer(ctx context.Context, containerID string) error
    StopContainer(ctx context.Context, containerID string) error
    RestartContainer(ctx context.Context, containerID string) error
    PauseContainer(ctx context.Context, containerID string) error
    UnpauseContainer(ctx context.Context, containerID string) error
    RemoveContainer(ctx context.Context, containerID string, force bool) error
    GetContainerStats(ctx context.Context, containerID string) (*models.ContainerStats, error)
    GetContainerLogs(ctx context.Context, containerID string, lines int) ([]string, error)
    Ping(ctx context.Context) error
    Close() error
}
```

Todos os métodos recebem `context.Context` como primeiro argumento para suportar cancelamento.

### 3.2 Implementação Real

**`struct Client`** wrapping `*client.Client` do Docker SDK.

**`NewClient(host string) (DockerClient, error)`:**
- Se `host == ""`, usa socket padrão do sistema
- Fazer ping após conectar, retornar erro descritivo se falhar

**`ListContainers`:**
- Usar `container.ListOptions{All: !onlyRunning}`
- Para cada container em estado `running`, buscar stats (pode ser em goroutine com WaitGroup)
- Buscar logs dos últimos 100 lines para todos
- Converter de `types.Container` para `models.Container`

**Parsing de CPU:**
```
cpuDelta = CPUStats.TotalUsage - PreCPUStats.TotalUsage
systemDelta = CPUStats.SystemCPUUsage - PreCPUStats.SystemCPUUsage
cpuPercent = (cpuDelta / systemDelta) * numCPUs * 100.0
```

**Parsing de logs Docker:**
- O Docker usa um protocolo multiplexado: cada frame tem 8 bytes de header
- Bytes 0: stream type (1=stdout, 2=stderr)
- Bytes 4-7: tamanho da mensagem em big-endian
- Implementar fallback para quando o header não estiver presente

**`convertContainer` deve mapear corretamente:**
- `container.Names[0]` sem o prefixo `/`
- Detectar health status a partir do campo `Status` do container
- `Started` deve vir de `container.Created` para containers que só têm esse dado na listagem; enriquecer com `InspectContainer` para dados de `StartedAt` precisos

### 3.3 Mock Client

Criar `internal/docker/mock.go` com `MockClient` que implementa `DockerClient` usando dados de `models.MockContainers()`. Útil para desenvolvimento sem Docker rodando.

---

## Etapa 5 — TUI Core

**Objetivo:** Aplicação TUI funcional com layout, navegação e auto-refresh.

### 5.1 Config da App

```go
type Config struct {
    Server      string
    Group       string
    RefreshRate time.Duration // default: 3s
    MaxLogLines int           // default: 1000
    DockerHost  string        // default: "" (socket padrão)
    UseMock     bool          // usar dados mock em vez de Docker real
    Session     *models.Session // sessão autenticada
}
```

### 5.2 Visual Completo do `kern see` (Modo Containers)

Esta é a especificação visual definitiva da tela principal. Implementar respeitando exatamente estas proporções e elementos:

```
╔════════════════════════════════════════════════════════════════════════════════╗
║  Server: ws://monitoring.acme.com | john_doe [admin] | 15:04:05 | ● Connected ║
╠═════════════════════════╦══════════════════════════════════════════════════════╣
║ Containers (7 total)    ║  Overview > Stats < Network  Storage  Logs          ║
║─────────────────────────║──────────────────────────────────────────────────────║
║ 📂 nginx (3)  2 running ║                                                      ║
║   ▶ nginx-web-1   8080  ║  Container Information                               ║
║   ▶ nginx-web-2   8081  ║                                                      ║
║   ■ nginx-web-3   exited║  Identity                                            ║
║                         ║    ID       : a1b2c3d4ef56                           ║
║ ▶ postgres-db    5432   ║    Name     : nginx-web-1                            ║
║ ▶ redis-cache    6379   ║    Image    : nginx                                  ║
║ ■ app-worker     exited ║    Tag      : latest                                 ║
║                         ║                                                      ║
║                         ║  Status                                              ║
║                         ║    Status   : ▶ running                              ║
║                         ║    Health   : ✓ healthy                              ║
║                         ║    Exit Code: 0 (success)                            ║
║                         ║                                                      ║
║                         ║  Timing                                              ║
║                         ║    Created  : 2026-03-23 13:00:00                    ║
║                         ║    Uptime   : 2h 4m                                  ║
║                         ║                                                      ║
║                         ║  Quick Stats                                         ║
║                         ║    CPU      : 5.2%                                   ║
║                         ║    Memory   : 50.0MB (9.8%)                          ║
║                         ║    Network  : ↓ 100MB  ↑ 200MB                      ║
╠═════════════════════════╩══════════════════════════════════════════════════════╣
║ [Tab]Foco  [s]Start  [t]Stop  [p]Pause  [d]Remove  [r]Refresh  [i]Perfil [q]Sair ║
╚════════════════════════════════════════════════════════════════════════════════╝
```

**Regras de cores:**
- Header: fundo azul escuro, texto branco/amarelo
- Status bar: fundo cinza escuro, atalhos em amarelo
- Container running: ícone e texto em verde `[green]`
- Container exited/dead: ícone e texto em vermelho `[red]`
- Container paused: ícone e texto em amarelo `[yellow]`
- Item selecionado na lista: fundo azul escuro (comportamento padrão do tview.List)
- Títulos de seção nos detalhes: amarelo `[yellow]`
- Valores normais: branco `[white]`
- Valores N/A: cinza `[gray]`
- Aba ativa: `> NomeAba <` em branco bright
- Abas inativas: nome em cinza
- Indicador de conexão: `●` verde se conectado, `●` vermelho se desconectado

**Regras de layout:**
- Usar `tview.Grid` com borders habilitados
- Coluna esquerda (containers): largura fixa de **42 colunas**
- Coluna direita (detalhes): expansível, ocupa o restante
- Header: altura fixa de **3 linhas**
- Status bar: altura fixa de **1 linha**
- Caracteres de borda: usar os do tview por padrão (não desenhar manualmente)

### 5.3 Visual do `kern see` (Modo Máquinas)

```
╔════════════════════════════════════════════════════════════════════════════════╗
║  Server: ws://monitoring.acme.com | john_doe [admin] | 15:04:05 | ● Connected ║
╠═════════════════════════╦══════════════════════════════════════════════════════╣
║ Machines (10 total)     ║  Overview  Resources  Processes                     ║
║─────────────────────────║──────────────────────────────────────────────────────║
║ 📁 frontend (2)         ║                                                      ║
║   ● web-server-01       ║  Machine Information                                 ║
║       CPU: 23.5%        ║                                                      ║
║   ● web-server-02       ║  Identity                                            ║
║       CPU: 55.3%        ║    Name     : web-server-01                          ║
║                         ║    IP       : 192.168.0.10                           ║
║ 📁 database (2)         ║    Group    : frontend                               ║
║   ● db-server-01        ║    Status   : ● online                               ║
║   ○ db-server-02        ║                                                      ║
║                         ║  Resources                                           ║
║ 📁 cache (2)            ║    CPU      : ████████░░░░░░░░░░░░░░░░░░░░ 23.5%    ║
║   ● cache-01            ║    Memory   : ████████████░░░░░░░░░░░░░░░░ 50.0%    ║
║   ○ cache-02            ║               4.0GB / 8.0GB                          ║
║                         ║    Disk     : ████████████████████░░░░░░░░ 46.9%    ║
║                         ║               120GB / 256GB                          ║
║                         ║                                                      ║
║                         ║  Uptime     : 2 days, 0 hours                        ║
║                         ║  Last Seen  : 2m ago                                 ║
╠═════════════════════════╩══════════════════════════════════════════════════════╣
║ [Tab]Foco  [r]Refresh  [i]Perfil  [q]Sair                                      ║
╚════════════════════════════════════════════════════════════════════════════════╝
```

**Regras de cores nas máquinas:**
- `●` verde: online
- `○` vermelho: offline
- `◐` amarelo: error

### 5.4 Layout do Grid

Usar `tview.Grid` com:
- Linhas: `[3, 0, 1]` (header fixo, conteúdo flexível, statusbar fixa)
- Colunas: `[42, 0]`

### 5.5 Ciclo de Vida da App

```
Run() {
    initializeDocker(ctx)
    initializeComponents()
    setupLayout()
    setupKeyBindings()
    startAutoRefresh(ctx)
    tviewApp.Run()
}
```

**`startAutoRefresh`:** Usar goroutine com `select` em `ticker.C` e `ctx.Done()`. Usar `context.WithCancel` para poder parar no shutdown.

**`performRefresh`:**
1. Guardar ID do container selecionado
2. Chamar `docker.ListContainers` com contexto
3. Usar `QueueUpdateDraw` para atualizar UI
4. Restaurar seleção por ID após atualização

**Cleanup no `quit()`:**
1. Cancelar context
2. Parar ticker
3. Parar header (goroutine do relógio)
4. `docker.Close()`
5. `tviewApp.Stop()`

**IMPORTANTE:** Nunca chamar `os.Exit()` dentro da lógica da TUI. Propagar erros para cima.

### 5.6 Atalhos de Teclado Globais

| Tecla | Ação |
|---|---|
| `q` / `Escape` | Sair da aplicação |
| `Tab` | Alternar foco entre painel esquerdo e direito |
| `1` a `5` | Ir direto para aba do painel de detalhes |
| `F1-F5` | Alternativa para trocar abas |
| `s` | Start no container selecionado |
| `t` | Stop no container selecionado |
| `p` | Pause no container selecionado |
| `u` | Unpause no container selecionado |
| `d` | Remove container (só se parado) — com confirmação |
| `r` | Forçar refresh imediato |
| `i` | Abrir painel de perfil do usuário logado |
| `?` | Mostrar modal de ajuda com todos os atalhos |

### 5.7 Sincronização

- Toda escrita em dados compartilhados entre goroutines deve usar `sync.Mutex` ou ocorrer dentro de `QueueUpdateDraw`
- O campo `Stats` do container é atualizado em goroutine separada — proteger com mutex ou copiar o valor

---

## Etapa 6 — Componentes TUI

**Objetivo:** Implementar todos os componentes visuais da interface.

### 6.1 Header (`components/header.go`)

- `tview.TextView` com `DynamicColors(true)`
- Exibir: Server, Username + Role (se logado), Hora atual, Status de conexão
- Formato: `Server: {url} | {username} [{role}] | {hora} | {indicador}`
- Atualizar relógio a cada segundo via goroutine com ticker
- Método `Stop()` para encerrar a goroutine do relógio
- Método `SetConnected(bool)` para alterar indicador de status
- Método `SetUser(username, role string)` para atualizar dados do usuário
- Se sessão expira em menos de 1 hora: exibir `[yellow]⚠ Session expires in 45m[white]` no header

### 6.2 Status Bar (`components/statusbar.go`)

- `tview.TextView` na parte inferior
- Exibir os atalhos principais de forma compacta:
  ```
  [Tab]Foco  [s]Start  [t]Stop  [p]Pause  [d]Remove  [r]Refresh  [i]Perfil  [?]Ajuda  [q]Sair
  ```
- Texto estático, sem necessidade de atualização

### 6.3 Container List (`components/containers.go`)

**Agrupamento hierárquico:**
- Agrupar containers por prefixo comum do nome (ex: `nginx-web-1` e `nginx-web-2` formam o grupo `nginx-web`)
- Algoritmo: para cada container, extrair o prefixo removendo o último segmento separado por `-`; se outro container compartilha o prefixo, criar um grupo
- Grupos com só 1 container não criam nível hierárquico — mostrar direto
- Grupos iniciam **colapsados** (exceto se só tem 1 container)

**Interface visual:**
```
📁 nginx (3 containers, 2 running)
  ▶ nginx-web-1 (a1b2c3d4)
      running  Port: 8080
  ▶ nginx-web-2 (e5f6g7h8)
      running  Port: 8081
  ■ nginx-web-3 (i9j0k1l2)
      exited   Age: 2h
▶ postgres-db (abc12345)
    running  Port: 5432
```

**Teclado na lista:**
- `Enter` / click: expandir/colapsar grupo OU selecionar container
- `→`: expandir grupo selecionado
- `←`: colapsar grupo selecionado OU ir ao grupo pai
- `↑`/`↓`: navegar pelos itens

**Métodos públicos:**
- `UpdateContainersPreserveSelection(containers []*models.Container, selectedID string)`
- `GetSelectedContainer() *models.Container`
- `SetSelectedFunc(fn func(*models.Container))`

### 6.4 Details Panel (`components/details.go`)

- `tview.TextView` com `DynamicColors(true)` e `Scrollable(true)`
- 5 abas: Overview, Stats, Network, Storage, Logs
- Cabeçalho de abas com aba atual destacada: `  Overview  > Stats <  Network  `
- Título do painel: `" [nomeDaAba] — [nomeContainer] "`
- Estado vazio elegante quando nenhum container selecionado

**Teclado no painel:**
- `←`/`→`: navegar entre abas
- `r`: refresh logs (na aba Logs) OU restart container (nas outras abas)

### 6.5 Modal de Confirmação (`components/modal.go`)

Para ações destrutivas (remove, stop), exibir modal:
```
┌─────────────────────────────┐
│  Confirmar ação             │
│                             │
│  Remover container          │
│  "postgres-db"?             │
│                             │
│  [Confirmar]   [Cancelar]   │
└─────────────────────────────┘
```

Usar `tview.Modal` do tview ou implementar com `tview.Frame`.

### 6.6 Aba Overview (`details/overview.go`)

Seções:
1. **Identity**: ID (curto), Name, Image, Tag
2. **Status**: Status (colorido + ícone), State, Health, Exit Code
3. **Timing**: Created, Started, Age, Uptime
4. **Configuration**: Command (truncado), Restart Policy, PIDs
5. **Quick Stats**: CPU %, Memória (MB e %), Rede (↓x ↑y), PIDs
6. **Labels**: Grid com até 10 labels (key: value)

### 6.7 Aba Stats (`details/stats.go`)

**Visualizações com barras de progresso Unicode:**
- Usar `█` para preenchido, `░` para vazio
- Barra de 40 chars de largura
- Cor dinâmica: verde < 50%, amarelo < 80%, vermelho >= 80%

Seções:
1. **CPU**: barra de uso geral + barra de throttling (se > 0)
2. **Memory**: 3 barras — Total, RSS, Cache — com valores absolutos
3. **Network I/O**: barras RX e TX relativas entre si + valores em MB
4. **Block I/O**: barras Read e Write relativas entre si + valores
5. **Process Info**: contagem de PIDs

Cache de 2 segundos para evitar rerender desnecessário.

### 6.8 Aba Network (`details/network.go`)

Tabelas com bordas ASCII:
1. **Port Mappings**: Private | Public | Type | IP
2. **Networks**: Network Name | IP Address | Gateway | MAC
3. **Statistics**: RX/TX em bytes, pacotes, erros, drops

### 6.9 Aba Storage (`details/storage.go`)

1. **Mounts**: Source | Destination | Type | Mode (rw/ro)
2. **Block I/O Stats**: Read/Write bytes e operações

### 6.10 Aba Logs (`details/logs.go`)

**Parsing de cada linha de log:**
1. Remover códigos ANSI (com regex ou substituição manual)
2. Extrair timestamp ISO8601 se presente: `2006-01-02T15:04:05.999Z`
3. Detectar nível de log por palavras-chave no conteúdo
4. Colorir por nível: ERROR=vermelho, WARN=amarelo, INFO=ciano, DEBUG=cinza
5. Exibir: `  42  15:04:05.123  [mensagem colorida]`

**Detecção de nível:**
- ERROR: "error", "err", "fatal", "panic", "failed", "exception"
- WARN: "warn", "warning", "deprecated"  
- DEBUG: "debug", "dbg", "trace", "verbose"
- INFO: default para linhas com conteúdo suficiente

**Regras de exibição:**
- Mostrar últimas 50 linhas
- Se há mais, exibir aviso `... mostrando últimas 50 de 200 linhas ...`
- Footer: `Press 'r' to refresh | Scroll to navigate`
- Cache de 5 segundos (comparar por comprimento + último elemento)

---

## Etapa 7 — Coleta de Métricas Locais

**Objetivo:** Implementar o comando `send` para enviar métricas reais da máquina.

### 6.1 Coletor de Métricas (`internal/metrics/collector.go`)

Implementar em **pure Go** (sem dependências externas pesadas):

**CPU Usage:**
- Linux: ler `/proc/stat`, calcular delta entre duas leituras com 100ms de intervalo
- Windows/Mac: usar `runtime.NumCPU()` + estimativa via goroutines (ou aceitar dependência leve como `gopsutil`)

**Memória:**
- Linux: ler `/proc/meminfo` (MemTotal, MemAvailable)
- Retornar `Used = Total - Available`

**Disco:**
- Usar `syscall.Statfs` (Unix) para o path `/`
- Windows: usar `GetDiskFreeSpaceEx` via syscall

**Processos em escuta:**
- Linux: ler `/proc/net/tcp` e `/proc/net/tcp6`
- Cruzar com `/proc/<pid>/net/tcp` para descobrir nomes

**Struct de resultado:**
```go
type Snapshot struct {
    CPUPercent  float64
    MemoryUsed  int64
    MemoryTotal int64
    DiskUsed    int64
    DiskTotal   int64
    Processes   []ProcessInfo
    CollectedAt time.Time
}
```

### 6.2 Comando `send`

```
kern send --name "minha-maquina" --group "backend" --interval 5
```

**Comportamento:**
1. Validar config (`kern config` deve ter sido executado)
2. Coletar snapshot inicial e exibir no terminal
3. Loop a cada `--interval` segundos:
   - Coletar snapshot
   - Enviar para servidor via WebSocket ou HTTP
   - Exibir status no terminal: `✓ 15:04:05 | CPU: 23.5% | MEM: 4.1/8GB | DISK: 120/256GB`
4. Responder a `Ctrl+C` graciosamente (context cancel)
5. Mostrar mensagem de encerramento

---

## Etapa 8 — Machine Monitoring na TUI

**Objetivo:** Integrar o monitoramento de máquinas remotas na interface.

### 7.1 Modo Dual (Containers + Máquinas)

O comando `kern see` deve suportar dois modos:
- `kern see` → Mostra só containers Docker locais (modo atual)
- `kern see --machines` → Mostra painel de máquinas remotas à esquerda e detalhes à direita

### 7.2 Layout com Máquinas

```
┌───────────────────────────────────────────────────┐
│  Header                                           │
├──────────────┬────────────────────────────────────┤
│  Machines    │   Machine Details                  │
│  (lista)     │   • Status, CPU, RAM, Disk         │
│              │   • Processos em escuta             │
│              │   • Uptime, Last Seen               │
│              │   • Containers nesta máquina (futuro)│
├──────────────┴────────────────────────────────────┤
│  Status Bar                                       │
└───────────────────────────────────────────────────┘
```

### 7.3 Machine List (`components/machines.go`)

- Agrupar máquinas por `Group`
- Exibir status com cor + ícone
- Se offline, mostrar `Last seen: 2m ago`
- Se online, mostrar `CPU: 23.5%`

### 7.4 Machine Details

Painel de detalhes com abas:
1. **Overview**: Status, IP, Uptime, Group
2. **Resources**: Barras de CPU, RAM, Disco (mesmo visual dos containers)
3. **Processes**: Tabela Address | Port | Name

---

## Etapa 9 — Qualidade e Finalização

**Objetivo:** Tornar o projeto robusto, testável e pronto para distribuição.

### 9.1 Testes

- `internal/docker/mock.go` — Mock que implementa a interface `DockerClient`
- `internal/auth/mock.go` — Mock que implementa a interface `AuthClient`
- Testes unitários para:
  - `models/container.go` — métodos de formatação e cálculos
  - `models/machine.go` — métodos de formatação
  - `models/user.go` — `IsExpired()`, `FormatExpiry()`
  - `docker/client.go` — parsing de logs Docker, cálculo de CPU
  - `auth/session.go` — save, load, delete, IsSessionValid
  - `auth/client.go` — tratamento de erros HTTP (401, 403, 500)
  - `config/config.go` — validação de campos obrigatórios
  - `tui/components/details/formatter.go` — FormatBytes, FormatNumber
  - `tui/components/details/logs.go` — stripAnsiCodes, detectLogLevel, extractTimestamp
- Usar apenas `testing` stdlib (sem frameworks externos)

### 9.2 Tratamento de Erros

**Regras:**
- Nunca usar `_` para ignorar um `error` — sempre tratar ou propagar
- Erros de UI (refresh falhou, stats indisponíveis) devem ser mostrados na interface, não causar crash
- Erros fatais (Docker não encontrado, config inválida) devem imprimir mensagem clara e retornar código de saída não-zero
- Usar `fmt.Errorf("contexto: %w", err)` para wrapping

### 9.3 Graceful Shutdown

```go
ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer cancel()
```

- Todas as goroutines devem respeitar o contexto
- `docker.Close()` deve ser chamado via `defer` garantido

### 9.4 Makefile

```makefile
.PHONY: build run test lint clean

build:
    go build -o kernus ./...

run:
    go run . see

test:
    go test ./... -v -race

lint:
    golangci-lint run ./...

clean:
    rm -f kernus kernus.exe
```

### 9.5 `.gitignore`

```gitignore
# Configuração e sessão (contêm credenciais e tokens)
config.json
session.json

# Binários
kernus
kernus.exe

# Build artifacts
dist/

# IDE
.vscode/
.idea/
*.swp
```

---

## Regras Gerais de Implementação

1. **Context em tudo:** Toda operação com I/O deve receber `context.Context` e respeitar cancelamento
2. **Interface antes de implementação:** Definir interfaces para dependências externas (Docker, servidor) antes de implementar — facilita testes e troca de implementação
3. **Erros sempre tratados:** Zero `_` ignorando errors em código de produção
4. **Sem `os.Exit` fora do main:** Apenas `cmd/root.go` pode chamar `os.Exit`
5. **Goroutines sempre têm dono:** Toda goroutine deve ter um mecanismo de parada (channel, context)
6. **Dados compartilhados protegidos:** Qualquer campo acessado por múltiplas goroutines deve ter `sync.Mutex` ou ser acessado exclusivamente via `QueueUpdateDraw`
7. **Segredos nunca em log:** Password e Token jamais aparecem em logs ou output de erro
8. **Config nunca no projeto:** O `config.json` vive em `~/.config/kernus/`, nunca no diretório do repositório

---

## Ordem de Implementação Recomendada

```
Etapa 1 (fundação)
  → Etapa 2 (models: container + machine + user)
  → Etapa 3 (auth: session, client, TUI de login)
  → Etapa 4 (docker client com interface)
  → Etapa 5 (TUI core: layout com mock + visual)
  → Etapa 6.1 (header) → Etapa 6.3 (lista) → Etapa 6.6 (overview)
  → Etapa 6.7 (stats) → Etapa 6.8 (network) → Etapa 6.9 (storage)
  → Etapa 6.10 (logs) → Etapa 6.5 (modal) → Etapa 6.2 (statusbar)
  → Etapa 6.4 (perfil dentro da TUI)
  → Etapa 7 (send + coleta de métricas)
  → Etapa 8 (machines na TUI)
  → Etapa 9 (qualidade, testes, finalização)
```

Em cada etapa, validar com `go build ./...` antes de continuar.
