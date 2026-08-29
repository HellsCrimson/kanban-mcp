package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"kanban-mcp/internal/model"
	"kanban-mcp/internal/store"
)

type handlers struct {
	store *store.Store
}

func Register(mux *http.ServeMux, s *store.Store) {
	h := &handlers{store: s}
	mux.HandleFunc("GET /api/health", h.health)
	mux.HandleFunc("GET /api/projects", h.listProjects)
	mux.HandleFunc("POST /api/projects", h.createProject)
	mux.HandleFunc("GET /api/projects/{id}", h.getProject)
	mux.HandleFunc("PATCH /api/projects/{id}", h.updateProject)
	mux.HandleFunc("DELETE /api/projects/{id}", h.deleteProject)
	mux.HandleFunc("GET /api/projects/{id}/board", h.getBoard)
	mux.HandleFunc("POST /api/projects/{id}/columns", h.createColumn)
	mux.HandleFunc("PATCH /api/columns/{id}", h.updateColumn)
	mux.HandleFunc("DELETE /api/columns/{id}", h.deleteColumn)
	mux.HandleFunc("GET /api/projects/{id}/cards", h.listCards)
	mux.HandleFunc("POST /api/projects/{id}/cards", h.createCard)
	mux.HandleFunc("GET /api/cards/{id}", h.getCard)
	mux.HandleFunc("PATCH /api/cards/{id}", h.updateCard)
	mux.HandleFunc("POST /api/cards/{id}/move", h.moveCard)
	mux.HandleFunc("DELETE /api/cards/{id}", h.deleteCard)
	mux.HandleFunc("POST /api/cards/{id}/updates", h.addUpdate)
	mux.HandleFunc("GET /api/cards/{id}/updates", h.listUpdates)
	mux.HandleFunc("POST /api/relationships", h.addRelationship)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
	case errors.Is(err, store.ErrConflict):
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
	case errors.Is(err, store.ErrInvalid):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}
}

func decode(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func (h *handlers) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

type createProjectReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (h *handlers) createProject(w http.ResponseWriter, r *http.Request) {
	var req createProjectReq
	if err := decode(r, &req); err != nil {
		fail(w, store.ErrInvalid)
		return
	}
	p, err := h.store.CreateProject(r.Context(), req.Name, req.Description)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 201, p)
}

func (h *handlers) listProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := h.store.ListProjects(r.Context())
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, projects)
}

func (h *handlers) getProject(w http.ResponseWriter, r *http.Request) {
	p, err := h.store.GetProject(r.Context(), r.PathValue("id"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, p)
}

type patchProjectReq struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

func (h *handlers) updateProject(w http.ResponseWriter, r *http.Request) {
	var req patchProjectReq
	if err := decode(r, &req); err != nil {
		fail(w, store.ErrInvalid)
		return
	}
	p, err := h.store.UpdateProject(r.Context(), r.PathValue("id"), req.Name, req.Description)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, p)
}

func (h *handlers) deleteProject(w http.ResponseWriter, r *http.Request) {
	if err := h.store.DeleteProject(r.Context(), r.PathValue("id")); err != nil {
		fail(w, err)
		return
	}
	w.WriteHeader(204)
}

func (h *handlers) getBoard(w http.ResponseWriter, r *http.Request) {
	b, err := h.store.GetBoard(r.Context(), r.PathValue("id"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, b)
}

type createColumnReq struct {
	Name string `json:"name"`
}

func (h *handlers) createColumn(w http.ResponseWriter, r *http.Request) {
	var req createColumnReq
	if err := decode(r, &req); err != nil {
		fail(w, store.ErrInvalid)
		return
	}
	col, err := h.store.CreateColumn(r.Context(), r.PathValue("id"), req.Name)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 201, col)
}

type patchColumnReq struct {
	Name     *string `json:"name"`
	Position *int    `json:"position"`
}

func (h *handlers) updateColumn(w http.ResponseWriter, r *http.Request) {
	var req patchColumnReq
	if err := decode(r, &req); err != nil {
		fail(w, store.ErrInvalid)
		return
	}
	col, err := h.store.UpdateColumn(r.Context(), r.PathValue("id"), req.Name, nil, req.Position)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, col)
}

func (h *handlers) deleteColumn(w http.ResponseWriter, r *http.Request) {
	if err := h.store.DeleteColumn(r.Context(), r.PathValue("id")); err != nil {
		fail(w, err)
		return
	}
	w.WriteHeader(204)
}

func (h *handlers) listCards(w http.ResponseWriter, r *http.Request) {
	f := store.CardFilter{}
	q := r.URL.Query()
	if v := q.Get("column_id"); v != "" {
		f.ColumnID = &v
	}
	if v := q.Get("type"); v != "" {
		t := model.CardType(v)
		f.Type = &t
	}
	if v := q.Get("parent_id"); v != "" {
		f.ParentID = &v
	}
	cards, err := h.store.ListCards(r.Context(), r.PathValue("id"), f)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, cards)
}

type createCardReq struct {
	ColumnID    string         `json:"column_id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Complexity  int            `json:"complexity"`
	Priority    model.Priority `json:"priority"`
	Type        model.CardType `json:"type"`
	ParentID    *string        `json:"parent_id"`
}

func (h *handlers) createCard(w http.ResponseWriter, r *http.Request) {
	var req createCardReq
	if err := decode(r, &req); err != nil {
		fail(w, store.ErrInvalid)
		return
	}
	card, err := h.store.CreateCard(r.Context(), store.NewCard{
		ProjectID:   r.PathValue("id"),
		ColumnID:    req.ColumnID,
		Title:       req.Title,
		Description: req.Description,
		Complexity:  req.Complexity,
		Priority:    req.Priority,
		Type:        req.Type,
		ParentID:    req.ParentID,
	})
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 201, card)
}

func (h *handlers) getCard(w http.ResponseWriter, r *http.Request) {
	detail, err := h.store.GetCardDetail(r.Context(), r.PathValue("id"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, detail)
}

type patchCardReq struct {
	Title       *string         `json:"title"`
	Description *string         `json:"description"`
	Complexity  *int            `json:"complexity"`
	Priority    *model.Priority `json:"priority"`
	Type        *model.CardType `json:"type"`
}

func (h *handlers) updateCard(w http.ResponseWriter, r *http.Request) {
	var req patchCardReq
	if err := decode(r, &req); err != nil {
		fail(w, store.ErrInvalid)
		return
	}
	card, err := h.store.UpdateCard(r.Context(), r.PathValue("id"), store.CardUpdate{
		Title:       req.Title,
		Description: req.Description,
		Complexity:  req.Complexity,
		Priority:    req.Priority,
		Type:        req.Type,
	})
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, card)
}

type moveCardReq struct {
	ColumnID string `json:"column_id"`
	Position *int   `json:"position"`
}

func (h *handlers) moveCard(w http.ResponseWriter, r *http.Request) {
	var req moveCardReq
	if err := decode(r, &req); err != nil {
		fail(w, store.ErrInvalid)
		return
	}
	card, err := h.store.MoveCard(r.Context(), r.PathValue("id"), req.ColumnID, req.Position)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, card)
}

func (h *handlers) deleteCard(w http.ResponseWriter, r *http.Request) {
	if err := h.store.DeleteCard(r.Context(), r.PathValue("id")); err != nil {
		fail(w, err)
		return
	}
	w.WriteHeader(204)
}

type addUpdateReq struct {
	Content string `json:"content"`
}

func (h *handlers) addUpdate(w http.ResponseWriter, r *http.Request) {
	var req addUpdateReq
	if err := decode(r, &req); err != nil {
		fail(w, store.ErrInvalid)
		return
	}
	u, err := h.store.AddUpdate(r.Context(), r.PathValue("id"), req.Content)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 201, u)
}

func (h *handlers) listUpdates(w http.ResponseWriter, r *http.Request) {
	updates, err := h.store.ListUpdates(r.Context(), r.PathValue("id"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, updates)
}

type addRelationshipReq struct {
	SourceID string             `json:"source_id"`
	TargetID string             `json:"target_id"`
	Type     model.RelationType `json:"type"`
}

func (h *handlers) addRelationship(w http.ResponseWriter, r *http.Request) {
	var req addRelationshipReq
	if err := decode(r, &req); err != nil {
		fail(w, store.ErrInvalid)
		return
	}
	rel, err := h.store.AddRelationship(r.Context(), req.SourceID, req.TargetID, req.Type)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 201, rel)
}
