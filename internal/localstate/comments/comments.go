package comments

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const persistFilename = "gui-entity-comments.json"

type Comment struct {
	ID         string `json:"id"`
	EntityType string `json:"entityType"`
	EntityID   string `json:"entityId"`
	Author     string `json:"author"`
	Text       string `json:"text"`
	CreatedAt  string `json:"createdAt"`
}

type Service struct {
	mu       sync.RWMutex
	rootDir  string
	path     string
	comments map[string][]Comment
}

type persisted struct {
	Comments map[string][]Comment `json:"comments"`
}

func New(rootDir string) *Service {
	s := &Service{
		rootDir:  rootDir,
		path:     filepath.Join(rootDir, persistFilename),
		comments: map[string][]Comment{},
	}
	if err := s.load(); err != nil {
		s.comments = map[string][]Comment{}
	}
	return s
}

func (s *Service) SetServer(host string, port uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.path = filepath.Join(s.rootDir, fmt.Sprintf("gui-entity-comments-%s_%d.json", host, port))
	s.comments = map[string][]Comment{}
	_ = s.loadLocked()
}

func entityKey(entityType, entityID string) string {
	return strings.ToLower(strings.TrimSpace(entityType)) + ":" + strings.TrimSpace(entityID)
}

func (s *Service) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked()
}

func (s *Service) loadLocked() error {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var p persisted
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	if p.Comments != nil {
		s.comments = p.Comments
	}
	return nil
}

func (s *Service) persistLocked() error {
	data, err := json.MarshalIndent(persisted{Comments: s.comments}, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *Service) GetComments(entityType, entityID string) []Comment {
	key := entityKey(entityType, entityID)
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := s.comments[key]
	out := make([]Comment, len(list))
	copy(out, list)
	return out
}

func (s *Service) GetAllComments() map[string][]Comment {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string][]Comment, len(s.comments))
	for k, list := range s.comments {
		copyList := make([]Comment, len(list))
		copy(copyList, list)
		out[k] = copyList
	}
	return out
}

func cleanUsername(author string) string {
	author = strings.TrimSpace(author)
	if author == "" {
		return "Operator"
	}
	if idx := strings.Index(author, "@"); idx > 0 {
		author = author[:idx]
	}
	if idx := strings.Index(author, " "); idx > 0 {
		author = author[:idx]
	}
	author = strings.TrimSpace(author)
	if author == "" {
		return "Operator"
	}
	return author
}

func (s *Service) AddComment(entityType, entityID, author, text string) (Comment, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return Comment{}, fmt.Errorf("comment text cannot be empty")
	}

	c := Comment{
		ID:         uuid.New().String(),
		EntityType: strings.ToLower(strings.TrimSpace(entityType)),
		EntityID:   strings.TrimSpace(entityID),
		Author:     cleanUsername(author),
		Text:       text,
		CreatedAt:  nowString(),
	}

	key := entityKey(entityType, entityID)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.comments[key] = append(s.comments[key], c)

	sort.Slice(s.comments[key], func(i, j int) bool {
		return createdAtTime(s.comments[key][i]).Before(createdAtTime(s.comments[key][j]))
	})

	if err := s.persistLocked(); err != nil {
		return Comment{}, err
	}
	return c, nil
}

func (s *Service) SetNote(entityType, entityID, author, text string) (Comment, error) {
	text = strings.TrimSpace(text)
	key := entityKey(entityType, entityID)

	s.mu.Lock()
	defer s.mu.Unlock()

	list := s.comments[key]
	if len(list) == 0 {
		if text == "" {
			return Comment{}, nil
		}
		c := Comment{
			ID:         uuid.New().String(),
			EntityType: strings.ToLower(strings.TrimSpace(entityType)),
			EntityID:   strings.TrimSpace(entityID),
			Author:     cleanUsername(author),
			Text:       text,
			CreatedAt:  nowString(),
		}
		s.comments[key] = append(s.comments[key], c)
		if err := s.persistLocked(); err != nil {
			return Comment{}, err
		}
		return c, nil
	}

	lastIdx := len(list) - 1
	if text == "" {
		s.comments[key] = list[:lastIdx]
	} else {
		s.comments[key][lastIdx].Text = text
		s.comments[key][lastIdx].Author = cleanUsername(author)
		s.comments[key][lastIdx].CreatedAt = nowString()
	}

	if err := s.persistLocked(); err != nil {
		return Comment{}, err
	}
	if len(s.comments[key]) > 0 {
		return s.comments[key][len(s.comments[key])-1], nil
	}
	return Comment{}, nil
}

func nowString() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func createdAtTime(c Comment) time.Time {
	ts, err := time.Parse(time.RFC3339Nano, c.CreatedAt)
	if err == nil {
		return ts
	}
	ts, err = time.Parse(time.RFC3339, c.CreatedAt)
	if err == nil {
		return ts
	}
	return time.Time{}
}

func (s *Service) DeleteComment(commentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	found := false
	for k, list := range s.comments {
		newList := make([]Comment, 0, len(list))
		for _, c := range list {
			if c.ID == commentID {
				found = true
				continue
			}
			newList = append(newList, c)
		}
		if len(newList) == 0 {
			delete(s.comments, k)
		} else {
			s.comments[k] = newList
		}
	}

	if !found {
		return fmt.Errorf("comment %q not found", commentID)
	}
	return s.persistLocked()
}
