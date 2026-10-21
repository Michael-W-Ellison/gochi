package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Michael-W-Ellison/gochi/internal/core"
)

func TestBooksThroughServer(t *testing.T) {
	server := newTestServer(t)
	server.parental, _ = loadParentalControls("")
	if err := server.createPet("Mochi", core.SpeciesCat, "", false, false); err != nil {
		t.Fatal(err)
	}
	server.mu.Lock()
	id, slot := string(server.pet.ID), server.slot
	server.mu.Unlock()
	if _, err := server.saveNow(); err != nil { // The autosave writes the book
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	server.handleBooks(rec, httptest.NewRequest(http.MethodGet, "/api/books", nil))
	if !strings.Contains(rec.Body.String(), `"name":"Mochi"`) || !strings.Contains(rec.Body.String(), `"saved":true`) {
		t.Fatalf("The books: %s", rec.Body.String())
	}
	rec = httptest.NewRecorder()
	server.handleBooks(rec, httptest.NewRequest(http.MethodGet, "/api/books?id="+id, nil))
	if !strings.Contains(rec.Body.String(), "Mochi came home.") {
		t.Errorf("Mochi's book: %s", rec.Body.String())
	}
	if rec := post(t, server.handleDeleteBook, `{"id":"`+id+`"}`); rec.Code != http.StatusConflict {
		t.Errorf("Deleting the book of a pet that is still saved: %d", rec.Code)
	}

	// Deleting the pet keeps its book, unless asked
	if rec := post(t, server.handleDelete, `{"slot":"`+slot+`","confirm_name":"Mochi"}`); rec.Code != http.StatusOK {
		t.Fatalf("Deleting Mochi: %d %s", rec.Code, rec.Body.String())
	}
	if _, err := readBook(server.store.books, id); err != nil {
		t.Fatalf("The book should stay: %v", err)
	}
	if rec := post(t, server.handleBookCandle, `{"id":"`+id+`","lit":true}`); rec.Code != http.StatusOK {
		t.Errorf("Lighting a candle: %d", rec.Code)
	}
	if book, _ := readBook(server.store.books, id); !book.Candle {
		t.Error("The candle should be lit")
	}
	if rec := post(t, server.handleDeleteBook, `{"id":"`+id+`"}`); rec.Code != http.StatusOK {
		t.Errorf("Deleting the book: %d", rec.Code)
	}
	if _, err := readBook(server.store.books, id); err == nil {
		t.Error("The book should be gone")
	}
	if rec := post(t, server.handleDeleteBook, `{"id":"../household"}`); rec.Code != http.StatusNotFound {
		t.Errorf("A book ID that isn't a pet's: %d", rec.Code)
	}
}

func TestProjectsThroughServer(t *testing.T) {
	server := newTestServer(t)
	server.house = newTestHousehold(t, "")
	if err := server.createPet("Mochi", core.SpeciesCat, "", false, false); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	server.handleProjects(rec, httptest.NewRequest(http.MethodGet, "/api/projects", nil))
	var resp struct {
		Data []project `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil || len(resp.Data) != 5 {
		t.Fatalf("Projects: %s", rec.Body.String())
	}
	for _, p := range resp.Data {
		if p.Target <= 0 || p.Progress > p.Target || p.Detail == "" {
			t.Errorf("A project: %+v", p)
		}
	}
}

// readBook reads a book
func readBook(b *bookStore, id string) (petBook, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.readLocked(id)
}
