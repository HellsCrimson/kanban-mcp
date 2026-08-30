# Kanban MCP

A self-hosted kanban board with a REST API, a React frontend, and an [MCP](https://modelcontextprotocol.io) server over HTTP so LLMs can manage your boards.

- **Backend:** Go (single binary, SQLite via `modernc.org/sqlite`, no CGO)
- **Frontend:** React + Vite, embedded into the Go binary at build time
- **MCP:** streamable HTTP endpoint at `/mcp` (stateless)

## Features

- Multiple projects; each project starts with default columns **Backlog, Todo, In Progress, Review, Done** — custom columns can be added, renamed, reordered (drag the column header), and deleted
- Any number of columns can be marked as **done columns** (the ✓ in the column header): cards there count as completed for epic progress. New projects start with the last column marked
- Cards have a per-project number (`#1`, `#2`, …), title, description, complexity (1–10), and priority (`low` / `medium` / `high` / `urgent`)
- Two card types: regular **tasks** and **epics**. Epics can have sub-cards; sub-cards move independently in any column and the epic shows progress (`2/5`, where "done" means the sub-card sits in a done column)
- Drag & drop cards between columns and to any position within a column — a line shows where the card will land
- The board updates live over SSE: changes made in another tab or by an MCP client appear without a refresh
- Cards can have updates (comments) and be linked to other cards with relationships: `blocks`, `depends_on`, `relates_to`
- Everything is available through the REST API and as MCP tools

## Quick start

```sh
# build (builds the frontend into the binary, then compiles Go)
cd web && npm install && npm run build && cd ..
go build -o kanban .

# run
./kanban                 # http://localhost:8080, db in ./kanban.db
./kanban -addr :9000 -db /path/to/board.db
```

Open http://localhost:8080 for the UI.

### Development

```sh
cd web && npm run dev    # Vite dev server on :5173, proxies /api to :8080
./kanban                 # in another terminal
```

## MCP

The MCP server is served at `http://localhost:8080/mcp` (streamable HTTP, stateless — no session management needed). Point any MCP client at it:

```json
{
  "mcpServers": {
    "kanban": {
      "url": "http://localhost:8080/mcp"
    }
  }
}
```

### Tools

| Tool | Description |
| --- | --- |
| `list_projects` | List all projects |
| `create_project` | Create a project (initialized with the default columns) |
| `get_project` | Full board state: all columns and cards |
| `get_card` | One card incl. its updates and relationships; epics also include their sub-cards with column + done status |
| `list_cards` | Cards of a project, filterable by column / type / parent |
| `create_card` | Create a task or epic; `parent_id` creates a sub-card of an epic, `position` inserts it at a 0-based index |
| `update_card` | Update title, description, complexity, priority, type |
| `move_card` | Move a card to a column, optionally at a 0-based position |
| `add_column` / `update_column` / `delete_column` | Manage custom columns; `update_column` also sets `position` and `is_done` |
| `delete_card` | Delete a card (sub-cards of an epic go with it) |
| `add_update` / `list_updates` | Add a comment to a card / list its comments (oldest first) |
| `add_relationship` | Link two cards of the same project: `blocks`, `depends_on`, `relates_to` |

## REST API

```
GET    /api/health
GET    /api/events?project_id=        # SSE change stream; omit project_id for all events
GET    /api/projects                  POST /api/projects
GET    /api/projects/{id}             PATCH /api/projects/{id}      DELETE /api/projects/{id}
GET    /api/projects/{id}/board       # project + columns + cards in one call
POST   /api/projects/{id}/columns     # {"name": "..."}
PATCH  /api/columns/{id}              # {"name": "...", "position": 0, "is_done": true}
DELETE /api/columns/{id}              # 409 if the column has cards
GET    /api/projects/{id}/cards?column_id=&type=epic|task&parent_id=
POST   /api/projects/{id}/cards       # {"column_id","title","description","complexity","priority","type","parent_id","position"}
GET    /api/cards/{id}                # detail; epics include sub-cards
PATCH  /api/cards/{id}                # any of title/description/complexity/priority/type
POST   /api/cards/{id}/move           # {"column_id", "position"?}
DELETE /api/cards/{id}
POST   /api/cards/{id}/updates        # {"content": "..."}
GET    /api/cards/{id}/updates        # list, oldest first
POST   /api/relationships             # {"source_id", "target_id", "type": "blocks"|"depends_on"|"relates_to"}
DELETE /api/relationships/{id}        # remove a link (id from the card detail's relationships)
```

## Notes

- Data lives in a single SQLite file (WAL mode). Back it up by copying the file. Existing databases are migrated on startup: the last column of each project becomes its done column, which is what it implicitly was before.
- Card numbers are plain per-project counters and are never reused.
- An epic with sub-cards cannot be converted back to a task; delete the sub-cards first.
- Relationships only link cards within the same project; `relates_to` is symmetric (A↔B counts as one link). Deleting a card removes its updates and relationships.
- The MCP endpoint has permissive CORS so browser-based MCP clients can reach it.
