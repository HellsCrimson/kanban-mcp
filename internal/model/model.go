package model

import "time"

var DefaultColumns = []string{"Backlog", "Todo", "In Progress", "Review", "Done"}

type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type Column struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	Name      string `json:"name"`
	Position  int    `json:"position"`
}

type CardType string

const (
	CardTypeTask CardType = "task"
	CardTypeEpic CardType = "epic"
)

func (t CardType) Valid() bool {
	return t == CardTypeTask || t == CardTypeEpic
}

type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
	PriorityUrgent Priority = "urgent"
)

var Priorities = []Priority{PriorityLow, PriorityMedium, PriorityHigh, PriorityUrgent}

func (p Priority) Valid() bool {
	for _, v := range Priorities {
		if p == v {
			return true
		}
	}
	return false
}

type Card struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	ColumnID    string    `json:"column_id"`
	Number      int       `json:"number"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Complexity  int       `json:"complexity"`
	Priority    Priority  `json:"priority"`
	Type        CardType  `json:"type"`
	ParentID    *string   `json:"parent_id,omitempty"`
	Position    int       `json:"position"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type BoardCard struct {
	Card
	SubTotal int `json:"sub_total"`
	SubDone  int `json:"sub_done"`
}

type Subcard struct {
	Card       Card   `json:"card"`
	ColumnName string `json:"column_name"`
	Done       bool   `json:"done"`
}

type Update struct {
	ID        string    `json:"id"`
	CardID    string    `json:"card_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type RelationType string

const (
	RelBlocks    RelationType = "blocks"
	RelDependsOn RelationType = "depends_on"
	RelRelatesTo RelationType = "relates_to"
)

var RelationTypes = []RelationType{RelBlocks, RelDependsOn, RelRelatesTo}

func (t RelationType) Valid() bool {
	for _, v := range RelationTypes {
		if t == v {
			return true
		}
	}
	return false
}

type Relationship struct {
	ID        string       `json:"id"`
	SourceID  string       `json:"source_id"`
	TargetID  string       `json:"target_id"`
	Type      RelationType `json:"type"`
	CreatedAt time.Time    `json:"created_at"`
}

// RelatedCard is one end of a relationship as seen from a card in CardDetail.
// Direction is "outgoing" when the detail card is the source, "incoming" otherwise.
type RelatedCard struct {
	ID        string       `json:"id"`
	Number    int          `json:"number"`
	Title     string       `json:"title"`
	Type      RelationType `json:"type"`
	Direction string       `json:"direction"`
}

type CardDetail struct {
	Card          BoardCard     `json:"card"`
	Subcards      []Subcard     `json:"subcards,omitempty"`
	Updates       []Update      `json:"updates,omitempty"`
	Relationships []RelatedCard `json:"relationships,omitempty"`
}

type ColumnCards struct {
	Column Column      `json:"column"`
	Cards  []BoardCard `json:"cards"`
}

type Board struct {
	Project Project       `json:"project"`
	Columns []ColumnCards `json:"columns"`
}
