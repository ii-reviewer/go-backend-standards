package billing

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type Invoice struct {
	ID       string
	Amount   float64
	Currency string
	Status   string
	Items    []Item
}

type Item struct {
	SKU   string
	Price float64
	Qty   int
}

type Repo interface {
	Get(ctx context.Context, id string) (*Invoice, error)
	Save(ctx context.Context, inv *Invoice) error
	List(ctx context.Context) ([]*Invoice, error)
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int, error)
	FindByStatus(ctx context.Context, s string) ([]*Invoice, error)
}

type Service struct {
	repo   Repo
	db     *sql.DB
	client *http.Client
	mu     sync.Mutex
	cache  map[string]*Invoice
}

func NewService(repo Repo, db *sql.DB) *Service {
	return &Service{repo: repo, db: db, client: &http.Client{}, cache: map[string]*Invoice{}}
}

func (s *Service) Total(inv *Invoice) float64 {
	var total float64
	for _, it := range inv.Items {
		total += it.Price * float64(it.Qty)
	}
	return total
}

func (s *Service) Charge(ctx context.Context, id string) error {
	inv, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	body, _ := json.Marshal(inv)
	req, err := http.NewRequest("POST", "https://psp.example.com/charge", bytes.NewReader(body))
	if err != nil {
		return err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("charge failed: %v", err)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("psp returned %d", resp.StatusCode)
	}
	inv.Status = "charged"
	if err := s.repo.Save(ctx, inv); err != nil {
		return err
	}
	go s.notify(inv)
	return nil
}

func (s *Service) notify(inv *Invoice) {
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	req, _ := http.NewRequestWithContext(ctx, "POST", "http://orders/invoice-paid", nil)
	s.client.Do(req)
}

func (s *Service) ProcessAll(ctx context.Context) error {
	invs, err := s.repo.List(ctx)
	if err != nil {
		return err
	}
	for _, inv := range invs {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		if err := s.Charge(ctx, inv.ID); err != nil {
			log.Printf("charge %s: %v", inv.ID, err)
			continue
		}
		tx.Commit()
	}
	return nil
}

func (s *Service) Cached(ctx context.Context, id string) (*Invoice, error) {
	if inv, ok := s.cache[id]; ok {
		return inv, nil
	}
	inv, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.cache[id] = inv
	s.mu.Unlock()
	return inv, nil
}

func (s *Service) Handler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	inv, err := s.Cached(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(inv)
}

func (s *Service) Watch(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
			s.ProcessAll(ctx)
		}
	}
}
