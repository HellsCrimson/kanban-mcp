package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"kanban-mcp/internal/model"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
	ErrInvalid  = errors.New("invalid input")
)

const schema = `
CREATE TABLE IF NOT EXISTS projects (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	next_card_number INTEGER NOT NULL DEFAULT 1,
	created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS board_columns (
	id TEXT PRIMARY KEY,
	project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	position INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_board_columns_project ON board_columns(project_id, position);
CREATE TABLE IF NOT EXISTS cards (
	id TEXT PRIMARY KEY,
	project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
	column_id TEXT NOT NULL REFERENCES board_columns(id) ON DELETE CASCADE,
	number INTEGER NOT NULL,
	title TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	complexity INTEGER NOT NULL DEFAULT 1,
	priority TEXT NOT NULL DEFAULT 'medium',
	type TEXT NOT NULL DEFAULT 'task',
	parent_id TEXT REFERENCES cards(id) ON DELETE CASCADE,
	position INTEGER NOT NULL,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_cards_project ON cards(project_id);
CREATE INDEX IF NOT EXISTS idx_cards_column ON cards(column_id, position);
CREATE INDEX IF NOT EXISTS idx_cards_parent ON cards(parent_id);
`

type Store struct {
	db *sql.DB
	mu sync.Mutex
}

func Open(path string) (*Store, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func newID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func now() string { return time.Now().UTC().Format(time.RFC3339) }

const cardColumns = `c.id, c.project_id, c.column_id, c.number, c.title, c.description, c.complexity, c.priority, c.type, c.parent_id, c.position, c.created_at, c.updated_at`

func scanCard(row interface{ Scan(dest ...any) error }) (model.Card, error) {
	var c model.Card
	var parent sql.NullString
	var prio, typ, created, updated string
	err := row.Scan(&c.ID, &c.ProjectID, &c.ColumnID, &c.Number, &c.Title, &c.Description, &c.Complexity, &prio, &typ, &parent, &c.Position, &created, &updated)
	if err != nil {
		return c, err
	}
	c.Priority = model.Priority(prio)
	c.Type = model.CardType(typ)
	if parent.Valid {
		p := parent.String
		c.ParentID = &p
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339, created)
	c.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	return c, nil
}

func (s *Store) ListProjects(ctx context.Context) ([]model.Project, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, description, created_at FROM projects ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Project{}
	for rows.Next() {
		var p model.Project
		var created string
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &created); err != nil {
			return nil, err
		}
		p.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) CreateProject(ctx context.Context, name, description string) (*model.Project, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: project name is required", ErrInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := newID()
	ts := now()
	if _, err := s.db.ExecContext(ctx, `INSERT INTO projects (id, name, description, created_at) VALUES (?, ?, ?, ?)`, id, name, description, ts); err != nil {
		return nil, err
	}
	for i, colName := range model.DefaultColumns {
		if _, err := s.db.ExecContext(ctx, `INSERT INTO board_columns (id, project_id, name, position) VALUES (?, ?, ?, ?)`, newID(), id, colName, i); err != nil {
			return nil, err
		}
	}
	p := model.Project{ID: id, Name: name, Description: description}
	p.CreatedAt, _ = time.Parse(time.RFC3339, ts)
	return &p, nil
}

func (s *Store) GetProject(ctx context.Context, id string) (*model.Project, error) {
	var p model.Project
	var created string
	err := s.db.QueryRowContext(ctx, `SELECT id, name, description, created_at FROM projects WHERE id = ?`, id).Scan(&p.ID, &p.Name, &p.Description, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &p, nil
}

func (s *Store) UpdateProject(ctx context.Context, id string, name, description *string) (*model.Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if name != nil && strings.TrimSpace(*name) == "" {
		return nil, fmt.Errorf("%w: project name cannot be empty", ErrInvalid)
	}
	set := []string{}
	args := []any{}
	if name != nil {
		set = append(set, "name = ?")
		args = append(args, strings.TrimSpace(*name))
	}
	if description != nil {
		set = append(set, "description = ?")
		args = append(args, *description)
	}
	if len(set) == 0 {
		return s.GetProject(ctx, id)
	}
	args = append(args, id)
	res, err := s.db.ExecContext(ctx, `UPDATE projects SET `+strings.Join(set, ", ")+` WHERE id = ?`, args...)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, ErrNotFound
	}
	return s.GetProject(ctx, id)
}

func (s *Store) DeleteProject(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ListColumns(ctx context.Context, projectID string) ([]model.Column, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, project_id, name, position FROM board_columns WHERE project_id = ? ORDER BY position`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Column{}
	for rows.Next() {
		var c model.Column
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.Name, &c.Position); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) CreateColumn(ctx context.Context, projectID, name string) (*model.Column, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: column name is required", ErrInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.GetProject(ctx, projectID); err != nil {
		return nil, err
	}
	var maxPos sql.NullInt64
	if err := s.db.QueryRowContext(ctx, `SELECT MAX(position) FROM board_columns WHERE project_id = ?`, projectID).Scan(&maxPos); err != nil {
		return nil, err
	}
	pos := 0
	if maxPos.Valid {
		pos = int(maxPos.Int64) + 1
	}
	col := model.Column{ID: newID(), ProjectID: projectID, Name: name, Position: pos}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO board_columns (id, project_id, name, position) VALUES (?, ?, ?, ?)`, col.ID, col.ProjectID, col.Name, col.Position); err != nil {
		return nil, err
	}
	return &col, nil
}

func (s *Store) UpdateColumn(ctx context.Context, id string, name, position *string, pos *int) (*model.Column, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var col model.Column
	err := s.db.QueryRowContext(ctx, `SELECT id, project_id, name, position FROM board_columns WHERE id = ?`, id).Scan(&col.ID, &col.ProjectID, &col.Name, &col.Position)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if name != nil {
		n := strings.TrimSpace(*name)
		if n == "" {
			return nil, fmt.Errorf("%w: column name cannot be empty", ErrInvalid)
		}
		col.Name = n
	}
	if pos != nil {
		cols, err := s.ListColumns(ctx, col.ProjectID)
		if err != nil {
			return nil, err
		}
		if *pos < 0 || *pos >= len(cols) {
			return nil, fmt.Errorf("%w: position %d out of range (0-%d)", ErrInvalid, *pos, len(cols)-1)
		}
		ordered := make([]model.Column, 0, len(cols))
		for _, c := range cols {
			if c.ID != col.ID {
				ordered = append(ordered, c)
			}
		}
		entry := model.Column{ID: col.ID}
		ordered = append(ordered[:*pos], append([]model.Column{entry}, ordered[*pos:]...)...)
		for i, c := range ordered {
			if _, err := s.db.ExecContext(ctx, `UPDATE board_columns SET position = ? WHERE id = ?`, i, c.ID); err != nil {
				return nil, err
			}
		}
		col.Position = *pos
	}
	if name != nil {
		if _, err := s.db.ExecContext(ctx, `UPDATE board_columns SET name = ? WHERE id = ?`, col.Name, col.ID); err != nil {
			return nil, err
		}
	}
	return &col, nil
}

func (s *Store) DeleteColumn(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM cards WHERE column_id = ?`, id).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return fmt.Errorf("%w: column has %d cards", ErrConflict, n)
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM board_columns WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if a, _ := res.RowsAffected(); a == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) GetCard(ctx context.Context, id string) (*model.Card, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+cardColumns+` FROM cards c WHERE c.id = ?`, id)
	c, err := scanCard(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

type CardFilter struct {
	ColumnID *string
	Type     *model.CardType
	ParentID *string
}

func (s *Store) lastColumnID(ctx context.Context, projectID string) (string, error) {
	var id string
	err := s.db.QueryRowContext(ctx, `SELECT id FROM board_columns WHERE project_id = ? ORDER BY position DESC LIMIT 1`, projectID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return id, err
}

func (s *Store) subProgress(ctx context.Context, projectID string) (map[string][2]int, error) {
	m := map[string][2]int{}
	lastCol, err := s.lastColumnID(ctx, projectID)
	if err != nil || lastCol == "" {
		return m, nil
	}
	rows, err := s.db.QueryContext(ctx, `SELECT parent_id, COUNT(*), SUM(CASE WHEN column_id = ? THEN 1 ELSE 0 END) FROM cards WHERE project_id = ? AND parent_id IS NOT NULL GROUP BY parent_id`, lastCol, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var pid string
		var total int
		var done sql.NullInt64
		if err := rows.Scan(&pid, &total, &done); err != nil {
			return nil, err
		}
		m[pid] = [2]int{total, int(done.Int64)}
	}
	return m, rows.Err()
}

func (s *Store) ListCards(ctx context.Context, projectID string, f CardFilter) ([]model.BoardCard, error) {
	if _, err := s.GetProject(ctx, projectID); err != nil {
		return nil, err
	}
	q := `SELECT ` + cardColumns + ` FROM cards c WHERE c.project_id = ?`
	args := []any{projectID}
	if f.ColumnID != nil {
		q += ` AND c.column_id = ?`
		args = append(args, *f.ColumnID)
	}
	if f.Type != nil {
		q += ` AND c.type = ?`
		args = append(args, string(*f.Type))
	}
	if f.ParentID != nil {
		q += ` AND c.parent_id = ?`
		args = append(args, *f.ParentID)
	}
	q += ` ORDER BY c.position`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.BoardCard{}
	for rows.Next() {
		c, err := scanCard(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, model.BoardCard{Card: c})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	progress, err := s.subProgress(ctx, projectID)
	if err != nil {
		return nil, err
	}
	for i := range out {
		if out[i].Type == model.CardTypeEpic {
			if p, ok := progress[out[i].ID]; ok {
				out[i].SubTotal = p[0]
				out[i].SubDone = p[1]
			}
		}
	}
	return out, nil
}

type NewCard struct {
	ProjectID   string
	ColumnID    string
	Title       string
	Description string
	Complexity  int
	Priority    model.Priority
	Type        model.CardType
	ParentID    *string
}

func (s *Store) CreateCard(ctx context.Context, in NewCard) (*model.BoardCard, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, fmt.Errorf("%w: title is required", ErrInvalid)
	}
	cardType := in.Type
	if cardType == "" {
		cardType = model.CardTypeTask
	}
	if !cardType.Valid() {
		return nil, fmt.Errorf("%w: type must be task or epic", ErrInvalid)
	}
	priority := in.Priority
	if priority == "" {
		priority = model.PriorityMedium
	}
	if !priority.Valid() {
		return nil, fmt.Errorf("%w: priority must be one of low, medium, high, urgent", ErrInvalid)
	}
	complexity := in.Complexity
	if complexity <= 0 {
		complexity = 1
	}
	if complexity > 10 {
		complexity = 10
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var projectID string
	err := s.db.QueryRowContext(ctx, `SELECT id FROM projects WHERE id = ?`, in.ProjectID).Scan(&projectID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: project %s", ErrNotFound, in.ProjectID)
	}
	if err != nil {
		return nil, err
	}

	columnID := in.ColumnID
	if columnID == "" {
		err = s.db.QueryRowContext(ctx, `SELECT id FROM board_columns WHERE project_id = ? ORDER BY position LIMIT 1`, projectID).Scan(&columnID)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: project has no columns", ErrInvalid)
		}
		if err != nil {
			return nil, err
		}
	} else {
		var colProject string
		err = s.db.QueryRowContext(ctx, `SELECT project_id FROM board_columns WHERE id = ?`, columnID).Scan(&colProject)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: column %s", ErrNotFound, columnID)
		}
		if err != nil {
			return nil, err
		}
		if colProject != projectID {
			return nil, fmt.Errorf("%w: column does not belong to project", ErrInvalid)
		}
	}

	if in.ParentID != nil {
		var parentType string
		err = s.db.QueryRowContext(ctx, `SELECT type FROM cards WHERE id = ? AND project_id = ?`, *in.ParentID, projectID).Scan(&parentType)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: parent card %s", ErrNotFound, *in.ParentID)
		}
		if err != nil {
			return nil, err
		}
		if parentType != string(model.CardTypeEpic) {
			return nil, fmt.Errorf("%w: parent card must be an epic", ErrInvalid)
		}
		cardType = model.CardTypeTask
	}

	var maxPos sql.NullInt64
	if err := s.db.QueryRowContext(ctx, `SELECT MAX(position) FROM cards WHERE column_id = ?`, columnID).Scan(&maxPos); err != nil {
		return nil, err
	}
	pos := 0
	if maxPos.Valid {
		pos = int(maxPos.Int64) + 1
	}

	var number int
	if err := s.db.QueryRowContext(ctx, `SELECT next_card_number FROM projects WHERE id = ?`, projectID).Scan(&number); err != nil {
		return nil, err
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE projects SET next_card_number = ? WHERE id = ?`, number+1, projectID); err != nil {
		return nil, err
	}

	id := newID()
	ts := now()
	var parentArg any
	if in.ParentID != nil {
		parentArg = *in.ParentID
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO cards (id, project_id, column_id, number, title, description, complexity, priority, type, parent_id, position, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, projectID, columnID, number, title, in.Description, complexity, string(priority), string(cardType), parentArg, pos, ts, ts)
	if err != nil {
		return nil, err
	}

	c, err := s.GetCard(ctx, id)
	if err != nil {
		return nil, err
	}
	return &model.BoardCard{Card: *c}, nil
}

func (s *Store) GetCardDetail(ctx context.Context, id string) (*model.CardDetail, error) {
	c, err := s.GetCard(ctx, id)
	if err != nil {
		return nil, err
	}
	detail := &model.CardDetail{Card: model.BoardCard{Card: *c}}
	if c.Type == model.CardTypeEpic {
		lastCol, _ := s.lastColumnID(ctx, c.ProjectID)
		progress, err := s.subProgress(ctx, c.ProjectID)
		if err != nil {
			return nil, err
		}
		if p, ok := progress[id]; ok {
			detail.Card.SubTotal = p[0]
			detail.Card.SubDone = p[1]
		}
		rows, err := s.db.QueryContext(ctx, `SELECT `+cardColumns+`, col.name FROM cards c JOIN board_columns col ON col.id = c.column_id WHERE c.parent_id = ? ORDER BY c.position, c.number`, id)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		detail.Subcards = []model.Subcard{}
		for rows.Next() {
			var sc model.Card
			var parent sql.NullString
			var prio, typ, created, updated, colName string
			if err := rows.Scan(&sc.ID, &sc.ProjectID, &sc.ColumnID, &sc.Number, &sc.Title, &sc.Description, &sc.Complexity, &prio, &typ, &parent, &sc.Position, &created, &updated, &colName); err != nil {
				return nil, err
			}
			sc.Priority = model.Priority(prio)
			sc.Type = model.CardType(typ)
			if parent.Valid {
				p := parent.String
				sc.ParentID = &p
			}
			sc.CreatedAt, _ = time.Parse(time.RFC3339, created)
			sc.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
			detail.Subcards = append(detail.Subcards, model.Subcard{Card: sc, ColumnName: colName, Done: lastCol != "" && sc.ColumnID == lastCol})
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	return detail, nil
}

type CardUpdate struct {
	Title       *string
	Description *string
	Complexity  *int
	Priority    *model.Priority
	Type        *model.CardType
}

func (s *Store) UpdateCard(ctx context.Context, id string, u CardUpdate) (*model.Card, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, err := s.GetCard(ctx, id)
	if err != nil {
		return nil, err
	}
	if u.Title != nil && strings.TrimSpace(*u.Title) == "" {
		return nil, fmt.Errorf("%w: title cannot be empty", ErrInvalid)
	}
	if u.Complexity != nil && (*u.Complexity < 1 || *u.Complexity > 10) {
		return nil, fmt.Errorf("%w: complexity must be between 1 and 10", ErrInvalid)
	}
	if u.Priority != nil && !u.Priority.Valid() {
		return nil, fmt.Errorf("%w: priority must be one of low, medium, high, urgent", ErrInvalid)
	}
	if u.Type != nil {
		if !u.Type.Valid() {
			return nil, fmt.Errorf("%w: type must be task or epic", ErrInvalid)
		}
		if *u.Type == model.CardTypeTask && cur.Type == model.CardTypeEpic {
			var n int
			if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM cards WHERE parent_id = ?`, id).Scan(&n); err != nil {
				return nil, err
			}
			if n > 0 {
				return nil, fmt.Errorf("%w: epic has %d sub-cards", ErrConflict, n)
			}
		}
	}
	set := []string{}
	args := []any{}
	if u.Title != nil {
		set = append(set, "title = ?")
		args = append(args, strings.TrimSpace(*u.Title))
	}
	if u.Description != nil {
		set = append(set, "description = ?")
		args = append(args, *u.Description)
	}
	if u.Complexity != nil {
		set = append(set, "complexity = ?")
		args = append(args, *u.Complexity)
	}
	if u.Priority != nil {
		set = append(set, "priority = ?")
		args = append(args, string(*u.Priority))
	}
	if u.Type != nil {
		set = append(set, "type = ?")
		args = append(args, string(*u.Type))
	}
	if len(set) > 0 {
		set = append(set, "updated_at = ?")
		args = append(args, now())
		args = append(args, id)
		if _, err := s.db.ExecContext(ctx, `UPDATE cards SET `+strings.Join(set, ", ")+` WHERE id = ?`, args...); err != nil {
			return nil, err
		}
	}
	return s.GetCard(ctx, id)
}

func (s *Store) MoveCard(ctx context.Context, id, toColumnID string, position *int) (*model.Card, error) {
	if toColumnID == "" {
		return nil, fmt.Errorf("%w: column_id is required", ErrInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	var cardProject, fromColumn string
	err := s.db.QueryRowContext(ctx, `SELECT project_id, column_id FROM cards WHERE id = ?`, id).Scan(&cardProject, &fromColumn)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var colProject string
	err = s.db.QueryRowContext(ctx, `SELECT project_id FROM board_columns WHERE id = ?`, toColumnID).Scan(&colProject)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: column %s", ErrNotFound, toColumnID)
	}
	if err != nil {
		return nil, err
	}
	if colProject != cardProject {
		return nil, fmt.Errorf("%w: column does not belong to the card's project", ErrInvalid)
	}

	rows, err := s.db.QueryContext(ctx, `SELECT id FROM cards WHERE column_id = ? AND id != ? ORDER BY position`, toColumnID, id)
	if err != nil {
		return nil, err
	}
	var ids []string
	for rows.Next() {
		var i string
		if err := rows.Scan(&i); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, i)
	}
	rows.Close()

	insertAt := len(ids)
	if position != nil {
		if *position < 0 || *position > len(ids) {
			return nil, fmt.Errorf("%w: position %d out of range (0-%d)", ErrInvalid, *position, len(ids))
		}
		insertAt = *position
	}
	if fromColumn != toColumnID {
		if _, err := s.db.ExecContext(ctx, `UPDATE cards SET column_id = ? WHERE id = ?`, toColumnID, id); err != nil {
			return nil, err
		}
	}
	ids = append(ids[:insertAt], append([]string{id}, ids[insertAt:]...)...)
	for i, cid := range ids {
		if _, err := s.db.ExecContext(ctx, `UPDATE cards SET position = ? WHERE id = ?`, i, cid); err != nil {
			return nil, err
		}
	}

	if fromColumn != toColumnID {
		rows2, err := s.db.QueryContext(ctx, `SELECT id FROM cards WHERE column_id = ? ORDER BY position`, fromColumn)
		if err != nil {
			return nil, err
		}
		var rest []string
		for rows2.Next() {
			var i string
			if err := rows2.Scan(&i); err != nil {
				rows2.Close()
				return nil, err
			}
			rest = append(rest, i)
		}
		rows2.Close()
		for i, cid := range rest {
			if _, err := s.db.ExecContext(ctx, `UPDATE cards SET position = ? WHERE id = ?`, i, cid); err != nil {
				return nil, err
			}
		}
	}

	return s.GetCard(ctx, id)
}

func (s *Store) DeleteCard(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `DELETE FROM cards WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) GetBoard(ctx context.Context, projectID string) (*model.Board, error) {
	p, err := s.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	cols, err := s.ListColumns(ctx, projectID)
	if err != nil {
		return nil, err
	}
	cards, err := s.ListCards(ctx, projectID, CardFilter{})
	if err != nil {
		return nil, err
	}
	byCol := map[string][]model.BoardCard{}
	for _, c := range cards {
		byCol[c.ColumnID] = append(byCol[c.ColumnID], c)
	}
	b := &model.Board{Project: *p, Columns: make([]model.ColumnCards, 0, len(cols))}
	for _, col := range cols {
		cc := model.ColumnCards{Column: col, Cards: byCol[col.ID]}
		if cc.Cards == nil {
			cc.Cards = []model.BoardCard{}
		}
		b.Columns = append(b.Columns, cc)
	}
	return b, nil
}
