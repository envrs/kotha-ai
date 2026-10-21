package symbol

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kothagpt/kotha/internal/db"
	"github.com/kothagpt/kotha/internal/pubsub"
)

// Symbol represents a code symbol extracted from a file via LSP.
type Symbol struct {
	ID            string
	SessionID     string
	FilePath      string
	Name          string
	Kind          string
	Range         string
	ContainerName string
	Language      string
	CreatedAt     int64
}

// Service persists and queries code symbols.
type Service interface {
	pubsub.Suscriber[Symbol]
	Create(ctx context.Context, sym Symbol) (Symbol, error)
	Get(ctx context.Context, id string) (Symbol, error)
	ListBySession(ctx context.Context, sessionID string) ([]Symbol, error)
	ListByFile(ctx context.Context, filePath, sessionID string) ([]Symbol, error)
	Search(ctx context.Context, sessionID, query string) ([]Symbol, error)
	Delete(ctx context.Context, id string) error
	DeleteSessionSymbols(ctx context.Context, sessionID string) error
	DeleteFileSymbols(ctx context.Context, filePath, sessionID string) error
}

type service struct {
	*pubsub.Broker[Symbol]
	q db.Querier
}

func NewService(q db.Querier) Service {
	return &service{
		Broker: pubsub.NewBroker[Symbol](),
		q:      q,
	}
}

func (s *service) Create(ctx context.Context, sym Symbol) (Symbol, error) {
	if sym.ID == "" {
		sym.ID = uuid.New().String()
	}
	if sym.CreatedAt == 0 {
		sym.CreatedAt = time.Now().UnixMilli()
	}
	var container sql.NullString
	if sym.ContainerName != "" {
		container = sql.NullString{String: sym.ContainerName, Valid: true}
	}
	dbSym, err := s.q.CreateSymbol(ctx, db.CreateSymbolParams{
		ID:            sym.ID,
		SessionID:     sym.SessionID,
		FilePath:      sym.FilePath,
		Name:          sym.Name,
		Kind:          sym.Kind,
		Range:         sym.Range,
		ContainerName: container,
		Language:      sym.Language,
	})
	if err != nil {
		return Symbol{}, err
	}
	created := s.fromDBItem(dbSym)
	s.Publish(pubsub.CreatedEvent, created)
	return created, nil
}

func (s *service) Get(ctx context.Context, id string) (Symbol, error) {
	dbSym, err := s.q.GetSymbol(ctx, id)
	if err != nil {
		return Symbol{}, err
	}
	return s.fromDBItem(dbSym), nil
}

func (s *service) ListBySession(ctx context.Context, sessionID string) ([]Symbol, error) {
	dbSyms, err := s.q.ListSymbolsBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	syms := make([]Symbol, len(dbSyms))
	for i, dbSym := range dbSyms {
		syms[i] = s.fromDBItem(dbSym)
	}
	return syms, nil
}

func (s *service) ListByFile(ctx context.Context, filePath, sessionID string) ([]Symbol, error) {
	dbSyms, err := s.q.ListSymbolsByFile(ctx, filePath, sessionID)
	if err != nil {
		return nil, err
	}
	syms := make([]Symbol, len(dbSyms))
	for i, dbSym := range dbSyms {
		syms[i] = s.fromDBItem(dbSym)
	}
	return syms, nil
}

func (s *service) Search(ctx context.Context, sessionID, query string) ([]Symbol, error) {
	pattern := "%" + strings.ToLower(query) + "%"
	dbSyms, err := s.q.SearchSymbols(ctx, sessionID, pattern)
	if err != nil {
		return nil, err
	}
	syms := make([]Symbol, len(dbSyms))
	for i, dbSym := range dbSyms {
		syms[i] = s.fromDBItem(dbSym)
	}
	return syms, nil
}

func (s *service) Delete(ctx context.Context, id string) error {
	sym, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.q.DeleteSymbol(ctx, sym.ID); err != nil {
		return err
	}
	s.Publish(pubsub.DeletedEvent, sym)
	return nil
}

func (s *service) DeleteSessionSymbols(ctx context.Context, sessionID string) error {
	syms, err := s.ListBySession(ctx, sessionID)
	if err != nil {
		return err
	}
	for _, sym := range syms {
		_ = s.q.DeleteSymbol(ctx, sym.ID)
	}
	return s.q.DeleteSessionSymbols(ctx, sessionID)
}

func (s *service) DeleteFileSymbols(ctx context.Context, filePath, sessionID string) error {
	return s.q.DeleteFileSymbols(ctx, filePath, sessionID)
}

func (s *service) fromDBItem(item db.Symbol) Symbol {
	return Symbol{
		ID:            item.ID,
		SessionID:     item.SessionID,
		FilePath:      item.FilePath,
		Name:          item.Name,
		Kind:          item.Kind,
		Range:         item.Range,
		ContainerName: item.ContainerName.String,
		Language:      item.Language,
		CreatedAt:     item.CreatedAt,
	}
}

// MarshalJSON implements json.Marshaler for Symbol.
func (s Symbol) MarshalJSON() ([]byte, error) {
	type alias Symbol
	return json.Marshal(alias(s))
}

// UnmarshalJSON implements json.Unmarshaler for Symbol.
func (s *Symbol) UnmarshalJSON(data []byte) error {
	type alias Symbol
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*s = Symbol(a)
	return nil
}