# Outlook MCP Server Local en Go — MVP

## Objetivo

Construir un MCP Server local utilizando Go que permita a herramientas como:

- OpenCode
- Claude Desktop
- Cursor
- Codex
- VSCode AI Agents

interactuar con Microsoft Outlook instalado localmente en Windows.

El objetivo principal es exponer herramientas (tools MCP) para:

- consultar correos
- consultar calendario
- crear borradores
- gestionar adjuntos
- automatizar tareas locales

sin depender inicialmente de Microsoft Graph ni de servicios cloud externos.

---

# Visión General

```txt
+----------------------+
| MCP Client           |
| (Cursor/OpenCode)    |
+----------+-----------+
           |
           | stdio
           v
+----------------------+
| Outlook MCP Server   |
|        Go            |
+----------+-----------+
           |
           | COM Automation
           v
+----------------------+
| Outlook Desktop      |
| Windows Local        |
+----------------------+
```

---

# Alcance MVP

## Incluido

### Correos

- Buscar correos
- Leer correos
- Crear borradores
- Enviar correos
- Listar adjuntos
- Descargar adjuntos

### Calendario

- Consultar eventos
- Crear eventos
- Crear reuniones
- Consultar disponibilidad básica

### Infraestructura

- MCP stdio
- Configuración YAML
- Logs
- Seguridad básica
- Windows only

---

# Estructura del Proyecto

```txt
outlook-mcp-go/
├─ cmd/
│  └─ outlook-mcp/
│     └─ main.go
│
├─ internal/
│  ├─ mcp/
│  │  ├─ server.go
│  │  ├─ tools.go
│  │  └─ handlers.go
│  │
│  ├─ outlook/
│  │  ├─ client.go
│  │  ├─ mail.go
│  │  ├─ calendar.go
│  │  └─ attachments.go
│  │
│  ├─ security/
│  │  └─ policy.go
│  │
│  ├─ config/
│  │  └─ config.go
│  │
│  └─ logging/
│     └─ logger.go
│
├─ configs/
│  └─ config.example.yaml
│
├─ README.md
├─ go.mod
└─ go.sum
```

---

# Configuración

```yaml
outlook:
  profile: "default"

security:
  allow_send_email: false
  allow_save_attachments: true

storage:
  attachments_dir: "C:\\OutlookMCP\\attachments"

logging:
  level: "info"

limits:
  max_results: 20
```

---

# Roadmap

## Fase 1 — MVP

- Inicializar servidor MCP
- Conexión COM
- Buscar correos
- Leer correos
- Consultar calendario

## Fase 2 — Productivo Local

- Adjuntos
- Borradores
- Envío de correo
- Auditoría
- Caché SQLite
- Mejor manejo de errores

## Fase 3 — Enterprise

- Microsoft Graph
- OAuth
- Exchange Online
- Multiusuario
- Permisos por herramienta
- Modo readonly
- Políticas empresariales

---

# Primer Objetivo

```txt
"Busca en Outlook correos sobre Kubernetes"
```

desde Cursor/OpenCode usando MCP local funcionando correctamente.
