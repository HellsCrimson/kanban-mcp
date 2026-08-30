package store

// Event is a change notification broadcast to subscribers so clients can
// refresh without polling.
type Event struct {
	// Type is "board" for anything inside a project (columns, cards, updates,
	// relationships) and "projects" when the project list itself changed.
	Type string `json:"type"`
	// ProjectID is the project the change belongs to.
	ProjectID string `json:"project_id"`
}

type subscriber struct {
	projectID string
	ch        chan Event
}

// Subscribe returns a channel of change events. An empty projectID receives
// every event; otherwise only events for that project are delivered. The
// returned function unsubscribes and closes the channel; it is safe to call
// more than once.
func (s *Store) Subscribe(projectID string) (<-chan Event, func()) {
	sub := &subscriber{projectID: projectID, ch: make(chan Event, 8)}
	s.subMu.Lock()
	s.subs[sub] = struct{}{}
	s.subMu.Unlock()

	var once bool
	return sub.ch, func() {
		s.subMu.Lock()
		defer s.subMu.Unlock()
		if once {
			return
		}
		once = true
		delete(s.subs, sub)
		close(sub.ch)
	}
}

// notify broadcasts an event. Sends are non-blocking: a subscriber that is not
// keeping up drops the event, which is harmless because every event triggers
// the same full reload on the client.
func (s *Store) notify(projectID, typ string) {
	s.subMu.Lock()
	defer s.subMu.Unlock()
	for sub := range s.subs {
		if sub.projectID != "" && sub.projectID != projectID {
			continue
		}
		select {
		case sub.ch <- Event{Type: typ, ProjectID: projectID}:
		default:
		}
	}
}
