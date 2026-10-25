package hub

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSealedSaves(t *testing.T) {
	save := []byte(`{"pet_data":{"name":"Biscuit"}}`)
	sealed, err := SealSave("account-key", "pet-key", save)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(sealed, []byte("Biscuit")) || !json.Valid(sealed) {
		t.Fatalf("A sealed save should be JSON without the save in it: %s", sealed)
	}
	if again, _ := SealSave("account-key", "pet-key", save); bytes.Equal(again, sealed) {
		t.Error("Each sealing should use a new nonce")
	}
	if opened, err := OpenSave("account-key", "pet-key", sealed); err != nil || !bytes.Equal(opened, save) {
		t.Errorf("Open: %s %v", opened, err)
	}
	if _, err := OpenSave("other-key", "pet-key", sealed); !errors.Is(err, ErrCannotOpen) {
		t.Errorf("Another account's key: %v", err)
	}
	if _, err := OpenSave("account-key", "other-pet", sealed); !errors.Is(err, ErrCannotOpen) {
		t.Errorf("A save moved to another pet: %v", err)
	}
	// A save that isn't encrypted is refused: anyone could have made it
	if opened, err := OpenSave("account-key", "pet-key", save); !errors.Is(err, ErrNotSealed) || opened != nil {
		t.Errorf("A plain save: %s %v", opened, err)
	}
}

func TestHubStoresOnlySealedSaves(t *testing.T) {
	h, server := testHub(t)
	alice := register(t, server)
	id, _ := h.Authenticate(alice.Token)
	key := PetKey("pet_1_Biscuit")
	if _, err := alice.UploadPetData(ctx, PetInfo{Key: key}, json.RawMessage(`{"secret":"tuna"}`), 0); err != nil {
		t.Fatal(err)
	}
	stored, err := os.ReadFile(filepath.Join(h.petDir(id), key+".json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(stored), "tuna") || !strings.Contains(string(stored), "gochi_sealed") {
		t.Errorf("The hub's file should hold only the sealed save: %s", stored)
	}
	if _, save, err := alice.DownloadPetData(ctx, key); err != nil || string(save) != `{"secret":"tuna"}` {
		t.Errorf("Download: %s %v", save, err)
	}
}
