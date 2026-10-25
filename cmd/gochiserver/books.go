package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
)

// Keepsake books: each pet's book of its life (internal/core/keepsakes.go)
// is written to <data dir>/books/<pet ID>.json whenever the pet is saved.
// Books stay on this computer and are never sent anywhere. A book stays
// after its pet is deleted, until the player deletes it too.

// bookIDPattern is what a pet ID used as a file name may hold
var bookIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,80}$`)

// errNoBook is returned for a book that doesn't exist
var errNoBook = errors.New("no such book")

// petBook is a pet's book
type petBook struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Species  string            `json:"species"`
	Breed    string            `json:"breed,omitempty"`
	Look     *core.PetLook     `json:"look,omitempty"`
	Outfit   map[string]string `json:"outfit,omitempty"`
	Alive    bool              `json:"alive"`
	Parents  []string          `json:"parents,omitempty"` // Pet IDs
	Pages    []core.Keepsake   `json:"pages"`
	Memorial *core.Memorial    `json:"memorial,omitempty"`
	Candle   bool              `json:"candle"` // The memorial candle is lit
	Updated  time.Time         `json:"updated"`
}

// bookStore keeps the books
type bookStore struct {
	mu  sync.Mutex
	dir string
}

func (b *bookStore) path(id string) (string, bool) {
	if b == nil || !bookIDPattern.MatchString(id) || reservedFileName(id) {
		return "", false
	}
	return filepath.Join(b.dir, id+".json"), true
}

// readLocked reads a book. Caller must hold b.mu.
func (b *bookStore) readLocked(id string) (petBook, error) {
	path, ok := b.path(id)
	if !ok {
		return petBook{}, errNoBook
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return petBook{}, errNoBook
	}
	if err != nil {
		return petBook{}, err
	}
	var book petBook
	if err := json.Unmarshal(data, &book); err != nil {
		return petBook{}, err
	}
	return book, nil
}

// writeLocked writes a book. Caller must hold b.mu.
func (b *bookStore) writeLocked(book petBook) error {
	path, ok := b.path(book.ID)
	if !ok {
		return errNoBook
	}
	if err := os.MkdirAll(b.dir, 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(book)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		_ = os.Remove(tmp) // Cut short (a full disk); the write's error is the one to report
		return err
	}
	return os.Rename(tmp, path)
}

// write brings a pet's book up to date with the pet, keeping its candle
func (b *bookStore) write(pet *core.DigitalPet) error {
	if b == nil {
		return nil
	}
	info := pet.BookInfo()
	b.mu.Lock()
	defer b.mu.Unlock()
	book := petBook{ID: string(pet.ID), Name: info.Name, Species: string(info.Species), Breed: info.Breed,
		Look: info.Look, Outfit: info.Outfit, Alive: info.Alive, Parents: info.Parents, Pages: pet.BookPages(),
		Memorial: pet.MemorialOf(), Updated: time.Now()}
	if old, err := b.readLocked(book.ID); err == nil {
		book.Candle = old.Candle
	}
	return b.writeLocked(book)
}

// list returns every book, newest first
func (b *bookStore) list() []petBook {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	entries, _ := os.ReadDir(b.dir)
	var books []petBook
	for _, e := range entries {
		id, ok := strings.CutSuffix(e.Name(), ".json")
		if !ok {
			continue
		}
		if book, err := b.readLocked(id); err == nil {
			books = append(books, book)
		}
	}
	sort.Slice(books, func(i, j int) bool { return books[i].Updated.After(books[j].Updated) })
	return books
}

// setCandle lights or puts out a memorial candle
func (b *bookStore) setCandle(id string, lit bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	book, err := b.readLocked(id)
	if err != nil {
		return err
	}
	book.Candle = lit
	return b.writeLocked(book)
}

// remove deletes a book
func (b *bookStore) remove(id string) error {
	path, ok := b.path(id)
	if !ok {
		return errNoBook
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// bookPage is a page as the page shows it
type bookPage struct {
	core.Keepsake
	Text    string                   `json:"text"`
	Wearing map[string]core.Cosmetic `json:"wearing,omitempty"` // The outfit, for drawing
}

// bookFamily is a relative in a book's family tree
type bookFamily struct {
	ID      string        `json:"id"`
	Name    string        `json:"name"`
	Species string        `json:"species"`
	Look    *core.PetLook `json:"look,omitempty"`
	Alive   bool          `json:"alive"`
}

// handleBooks handles GET /api/books - every book - and GET
// /api/books?id= - one book with its pages' text and its family tree
func (s *Server) handleBooks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}
	if s.store == nil {
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: []petBook{}})
		return
	}
	books := s.store.books.list()
	saved := map[string]bool{}
	if list, err := s.store.List(); err == nil {
		for _, p := range list {
			saved[p.ID] = true
		}
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		type summary struct {
			ID      string        `json:"id"`
			Name    string        `json:"name"`
			Species string        `json:"species"`
			Look    *core.PetLook `json:"look,omitempty"`
			Alive   bool          `json:"alive"`
			Pages   int           `json:"pages"`
			Saved   bool          `json:"saved"` // The pet is still saved (its book can't be deleted on its own)
		}
		list := []summary{}
		for _, b := range books {
			list = append(list, summary{ID: b.ID, Name: b.Name, Species: b.Species, Look: b.Look, Alive: b.Alive,
				Pages: len(b.Pages), Saved: saved[b.ID]})
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: list})
		return
	}

	i := slices.IndexFunc(books, func(b petBook) bool { return b.ID == id })
	if i < 0 {
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No such book"})
		return
	}
	book := books[i]
	byID := map[string]petBook{}
	for _, b := range books {
		byID[b.ID] = b
	}
	relative := func(id string) (bookFamily, bool) {
		b, ok := byID[id]
		return bookFamily{ID: b.ID, Name: b.Name, Species: b.Species, Look: b.Look, Alive: b.Alive}, ok
	}
	var parents, grandparents, children []bookFamily
	for _, pid := range book.Parents {
		if p, ok := relative(pid); ok {
			parents = append(parents, p)
			for _, gid := range byID[pid].Parents {
				if g, ok := relative(gid); ok {
					grandparents = append(grandparents, g)
				}
			}
		}
	}
	for _, b := range books {
		if slices.Contains(b.Parents, book.ID) {
			c, _ := relative(b.ID)
			children = append(children, c)
		}
	}
	pages := []bookPage{}
	for _, k := range book.Pages {
		if text := core.KeepsakeText(k, book.Name); text != "" {
			page := bookPage{Keepsake: k, Text: text, Wearing: map[string]core.Cosmetic{}}
			for slot, id := range k.Outfit {
				if c, ok := core.FindCosmetic(id); ok {
					page.Wearing[slot] = c
				}
			}
			pages = append(pages, page)
		}
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{
		"id": book.ID, "name": book.Name, "species": book.Species, "breed": book.Breed, "look": book.Look,
		"outfit": book.Outfit, "alive": book.Alive, "pages": pages, "memorial": book.Memorial, "candle": book.Candle,
		"saved": saved[book.ID], "parents": parents, "grandparents": grandparents, "children": children,
	}})
}

// handleBookCandle handles POST /api/books/candle with {"id", "lit"}
func (s *Server) handleBookCandle(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID  string `json:"id"`
		Lit bool   `json:"lit"`
	}
	if !decodePost(w, r, &req) {
		return
	}
	if s.store == nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No such book"})
		return
	}
	if err := s.store.books.setCandle(req.ID, req.Lit); err != nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No such book"})
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true})
}

// handleDeleteBook handles POST /api/books/delete with {"id",
// "parent_pin"}: deletes the book of a pet that is no longer saved, with
// the parent PIN if one is set, as deleting the pet needs it
func (s *Server) handleDeleteBook(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID        string `json:"id"`
		ParentPIN string `json:"parent_pin"`
	}
	if !decodePost(w, r, &req) {
		return
	}
	if s.store == nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No such book"})
		return
	}
	if !s.parentAllows(w, r, req.ParentPIN) {
		return
	}
	list, err := s.store.List()
	if err != nil { // Whether its pet is still saved can't be told
		writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Could not read the saved pets"})
		return
	}
	for _, p := range list {
		if p.ID == req.ID {
			writeJSON(w, http.StatusConflict, APIResponse{Success: false,
				Error: p.Name + " is still saved; its book can be deleted with it"})
			return
		}
	}
	if err := s.store.books.remove(req.ID); err != nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No such book"})
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true})
}
