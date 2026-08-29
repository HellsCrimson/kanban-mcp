package mcpserver

import (
	"context"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"kanban-mcp/internal/model"
	"kanban-mcp/internal/store"
)

func NewHandler(s *store.Store) http.Handler {
	srv := mcp.NewServer(&mcp.Implementation{Name: "kanban", Version: "1.0.0"}, nil)
	t := &tools{store: s}

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_projects",
		Description: "List all kanban projects with their id and name.",
	}, t.listProjects)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "create_project",
		Description: "Create a new project. It is initialized with the default columns: Backlog, Todo, In Progress, Review, Done.",
	}, t.createProject)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_project",
		Description: "Get a project's full board state: all columns and all cards in each column.",
	}, t.getProject)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_card",
		Description: "Get a single card. If the card is an epic, its sub-cards are included with their column name and done status.",
	}, t.getCard)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "add_column",
		Description: "Add a custom column to a project (appended after the last column).",
	}, t.addColumn)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "update_column",
		Description: "Rename a column and/or change its position. Position is the 0-based index among the project's columns.",
	}, t.updateColumn)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "delete_column",
		Description: "Delete a column. Fails if the column still contains cards.",
	}, t.deleteColumn)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_cards",
		Description: "List cards of a project, optionally filtered by column_id, type (epic or task) and/or parent_id (to list the sub-cards of an epic).",
	}, t.listCards)

	mcp.AddTool(srv, &mcp.Tool{
		Name: "create_card",
		Description: "Create a card in a project. Type is 'task' (default) or 'epic'. " +
			"To create a sub-card of an epic, set parent_id to the epic's id; sub-cards are always tasks and can be moved independently. " +
			"column_id defaults to the first column. complexity is 1-10 (default 1), priority is low|medium|high|urgent (default medium).",
	}, t.createCard)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "update_card",
		Description: "Update a card's title, description, complexity (1-10), priority (low|medium|high|urgent) and/or type. An epic that has sub-cards cannot be converted to a task.",
	}, t.updateCard)

	mcp.AddTool(srv, &mcp.Tool{
		Name: "move_card",
		Description: "Move a card to another column of the same project. Position is the 0-based index within the target column, " +
			"relative to the column's other cards (the moved card itself is not counted). Omit position to append at the end.",
	}, t.moveCard)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "delete_card",
		Description: "Delete a card. Sub-cards of an epic are deleted together with it.",
	}, t.deleteCard)

	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return srv }, &mcp.StreamableHTTPOptions{Stateless: true})
}

type tools struct {
	store *store.Store
}

func str(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func ip(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func prio(p *model.Priority) model.Priority {
	if p == nil {
		return ""
	}
	return *p
}

func ct(p *model.CardType) model.CardType {
	if p == nil {
		return ""
	}
	return *p
}

type listProjectsOut struct {
	Projects []model.Project `json:"projects"`
}

func (t *tools) listProjects(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, listProjectsOut, error) {
	projects, err := t.store.ListProjects(ctx)
	if projects == nil {
		projects = []model.Project{}
	}
	return nil, listProjectsOut{Projects: projects}, err
}

type createProjectIn struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

func (t *tools) createProject(ctx context.Context, _ *mcp.CallToolRequest, in createProjectIn) (*mcp.CallToolResult, *model.Project, error) {
	p, err := t.store.CreateProject(ctx, in.Name, in.Description)
	return nil, p, err
}

type idIn struct {
	ProjectID string `json:"project_id"`
}

func (t *tools) getProject(ctx context.Context, _ *mcp.CallToolRequest, in idIn) (*mcp.CallToolResult, *model.Board, error) {
	b, err := t.store.GetBoard(ctx, in.ProjectID)
	return nil, b, err
}

type cardIDIn struct {
	CardID string `json:"card_id"`
}

func (t *tools) getCard(ctx context.Context, _ *mcp.CallToolRequest, in cardIDIn) (*mcp.CallToolResult, *model.CardDetail, error) {
	d, err := t.store.GetCardDetail(ctx, in.CardID)
	return nil, d, err
}

type addColumnIn struct {
	ProjectID string `json:"project_id"`
	Name      string `json:"name"`
}

func (t *tools) addColumn(ctx context.Context, _ *mcp.CallToolRequest, in addColumnIn) (*mcp.CallToolResult, *model.Column, error) {
	c, err := t.store.CreateColumn(ctx, in.ProjectID, in.Name)
	return nil, c, err
}

type updateColumnIn struct {
	ColumnID string  `json:"column_id"`
	Name     *string `json:"name,omitempty"`
	Position *int    `json:"position,omitempty"`
}

func (t *tools) updateColumn(ctx context.Context, _ *mcp.CallToolRequest, in updateColumnIn) (*mcp.CallToolResult, *model.Column, error) {
	c, err := t.store.UpdateColumn(ctx, in.ColumnID, in.Name, nil, in.Position)
	return nil, c, err
}

type columnIDIn struct {
	ColumnID string `json:"column_id"`
}

func (t *tools) deleteColumn(ctx context.Context, _ *mcp.CallToolRequest, in columnIDIn) (*mcp.CallToolResult, map[string]string, error) {
	if err := t.store.DeleteColumn(ctx, in.ColumnID); err != nil {
		return nil, nil, err
	}
	return nil, map[string]string{"message": "column deleted"}, nil
}

type listCardsIn struct {
	ProjectID string          `json:"project_id"`
	ColumnID  *string         `json:"column_id,omitempty"`
	Type      *model.CardType `json:"type,omitempty"`
	ParentID  *string         `json:"parent_id,omitempty"`
}

type listCardsOut struct {
	Cards []model.BoardCard `json:"cards"`
}

func (t *tools) listCards(ctx context.Context, _ *mcp.CallToolRequest, in listCardsIn) (*mcp.CallToolResult, listCardsOut, error) {
	cards, err := t.store.ListCards(ctx, in.ProjectID, store.CardFilter{ColumnID: in.ColumnID, Type: in.Type, ParentID: in.ParentID})
	if cards == nil {
		cards = []model.BoardCard{}
	}
	return nil, listCardsOut{Cards: cards}, err
}

type createCardIn struct {
	ProjectID   string          `json:"project_id"`
	ColumnID    *string         `json:"column_id,omitempty"`
	Title       string          `json:"title"`
	Description *string         `json:"description,omitempty"`
	Complexity  *int            `json:"complexity,omitempty"`
	Priority    *model.Priority `json:"priority,omitempty"`
	Type        *model.CardType `json:"type,omitempty"`
	ParentID    *string         `json:"parent_id,omitempty"`
}

func (t *tools) createCard(ctx context.Context, _ *mcp.CallToolRequest, in createCardIn) (*mcp.CallToolResult, *model.BoardCard, error) {
	c, err := t.store.CreateCard(ctx, store.NewCard{
		ProjectID:   in.ProjectID,
		ColumnID:    str(in.ColumnID),
		Title:       in.Title,
		Description: str(in.Description),
		Complexity:  ip(in.Complexity),
		Priority:    prio(in.Priority),
		Type:        ct(in.Type),
		ParentID:    in.ParentID,
	})
	return nil, c, err
}

type updateCardIn struct {
	CardID      string          `json:"card_id"`
	Title       *string         `json:"title,omitempty"`
	Description *string         `json:"description,omitempty"`
	Complexity  *int            `json:"complexity,omitempty"`
	Priority    *model.Priority `json:"priority,omitempty"`
	Type        *model.CardType `json:"type,omitempty"`
}

func (t *tools) updateCard(ctx context.Context, _ *mcp.CallToolRequest, in updateCardIn) (*mcp.CallToolResult, *model.Card, error) {
	c, err := t.store.UpdateCard(ctx, in.CardID, store.CardUpdate{
		Title:       in.Title,
		Description: in.Description,
		Complexity:  in.Complexity,
		Priority:    in.Priority,
		Type:        in.Type,
	})
	return nil, c, err
}

type moveCardIn struct {
	CardID   string `json:"card_id"`
	ColumnID string `json:"column_id"`
	Position *int   `json:"position,omitempty"`
}

func (t *tools) moveCard(ctx context.Context, _ *mcp.CallToolRequest, in moveCardIn) (*mcp.CallToolResult, *model.Card, error) {
	c, err := t.store.MoveCard(ctx, in.CardID, in.ColumnID, in.Position)
	return nil, c, err
}

func (t *tools) deleteCard(ctx context.Context, _ *mcp.CallToolRequest, in cardIDIn) (*mcp.CallToolResult, map[string]string, error) {
	if err := t.store.DeleteCard(ctx, in.CardID); err != nil {
		return nil, nil, err
	}
	return nil, map[string]string{"message": "card deleted"}, nil
}
