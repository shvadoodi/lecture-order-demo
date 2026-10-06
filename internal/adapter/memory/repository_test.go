package memory

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	domain "github.com/shvadoodi/lecture-order-demo/internal/order"
)

func TestRepositoryIsolatesMutableOrderData(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryOrderRepository()
	now := time.Now().UTC()
	original := domain.Order{ID: "ORD-1", Items: []domain.OrderItem{{ProductID: "P-1", Quantity: 1}}, UpdatedAt: &now}
	if err := repo.Create(ctx, original); err != nil {
		t.Fatal(err)
	}
	assertStored := func() {
		t.Helper()
		stored, err := repo.GetByID(ctx, original.ID)
		if err != nil {
			t.Fatal(err)
		}
		if stored.Items[0].Quantity != 1 || stored.UpdatedAt.IsZero() {
			t.Fatalf("stored order was mutated: %+v", stored)
		}
	}
	original.Items[0].Quantity = 99
	*original.UpdatedAt = time.Time{}
	assertStored()
	fetched, err := repo.GetByID(ctx, original.ID)
	if err != nil {
		t.Fatal(err)
	}
	fetched.Items[0].Quantity = 99
	*fetched.UpdatedAt = time.Time{}
	assertStored()
	all, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	all[0].Items[0].Quantity = 99
	*all[0].UpdatedAt = time.Time{}
	assertStored()
	updated, err := repo.GetByID(ctx, original.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Update(ctx, updated); err != nil {
		t.Fatal(err)
	}
	updated.Items[0].Quantity = 99
	*updated.UpdatedAt = time.Time{}
	assertStored()
}

func TestRepositoryCRUD(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryOrderRepository()
	empty, err := repo.GetAll(ctx)
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("empty=%v err=%v", empty, err)
	}
	first := domain.Order{ID: "ORD-1", CreatedAt: time.Now().UTC(), Items: []domain.OrderItem{{ProductID: "P-1", Quantity: 1}}}
	second := domain.Order{ID: "ORD-2", CreatedAt: first.CreatedAt.Add(time.Minute)}
	if err := repo.Create(ctx, second); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, first); err != nil {
		t.Fatal(err)
	}
	all, err := repo.GetAll(ctx)
	if err != nil || len(all) != 2 || all[0].ID != first.ID || all[1].ID != second.ID {
		t.Fatalf("orders=%v err=%v", all, err)
	}
	first.CustomerID = "CUS-2"
	if err := repo.Update(ctx, first); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetByID(ctx, first.ID)
	if err != nil || got.CustomerID != first.CustomerID {
		t.Fatalf("order=%v err=%v", got, err)
	}
	if err := repo.Delete(ctx, first.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetByID(ctx, first.ID); !errors.Is(err, domain.ErrOrderNotFound) {
		t.Fatalf("get error=%v", err)
	}
	if err := repo.Update(ctx, first); !errors.Is(err, domain.ErrOrderNotFound) {
		t.Fatalf("update error=%v", err)
	}
	if err := repo.Delete(ctx, first.ID); !errors.Is(err, domain.ErrOrderNotFound) {
		t.Fatalf("delete error=%v", err)
	}
}

func TestRepositoryCanceledOperations(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	repo := NewInMemoryOrderRepository()
	original := domain.Order{ID: "ORD-1", CustomerID: "CUS-1"}
	if err := repo.Create(context.Background(), original); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		call func() error
	}{
		{"create", func() error { return repo.Create(ctx, domain.Order{ID: "ORD-2"}) }},
		{"get", func() error { _, err := repo.GetByID(ctx, "ORD-1"); return err }},
		{"list", func() error { _, err := repo.GetAll(ctx); return err }},
		{"update", func() error { return repo.Update(ctx, domain.Order{ID: "ORD-1", CustomerID: "changed"}) }},
		{"delete", func() error { return repo.Delete(ctx, "ORD-1") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.call(); !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		})
	}
	all, err := repo.GetAll(context.Background())
	if err != nil || len(all) != 1 || all[0].CustomerID != "CUS-1" {
		t.Fatalf("canceled operations changed storage: %v, %v", all, err)
	}
}

func TestRepositoryConcurrentAccess(t *testing.T) {
	repo := NewInMemoryOrderRepository()
	ctx := context.Background()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			record := domain.Order{ID: fmt.Sprintf("ORD-%d", i)}
			if err := repo.Create(ctx, record); err != nil {
				t.Error(err)
				return
			}
			if _, err := repo.GetByID(ctx, record.ID); err != nil {
				t.Error(err)
			}
			if _, err := repo.GetAll(ctx); err != nil {
				t.Error(err)
			}
			record.CustomerID = "updated"
			if err := repo.Update(ctx, record); err != nil {
				t.Error(err)
			}
			if err := repo.Delete(ctx, record.ID); err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	all, err := repo.GetAll(ctx)
	if err != nil || len(all) != 0 {
		t.Fatalf("orders=%v err=%v", all, err)
	}
}

func TestRepositoryPreservesNilAndEmptyItems(t *testing.T) {
	for _, items := range [][]domain.OrderItem{nil, {}} {
		repo := NewInMemoryOrderRepository()
		record := domain.Order{ID: "ORD-1", Items: items}
		if err := repo.Create(context.Background(), record); err != nil {
			t.Fatal(err)
		}
		got, err := repo.GetByID(context.Background(), record.ID)
		if err != nil || (got.Items == nil) != (items == nil) || got.UpdatedAt != nil {
			t.Fatalf("order=%+v err=%v", got, err)
		}
	}
}
