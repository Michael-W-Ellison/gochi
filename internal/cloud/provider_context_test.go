package cloud

import (
	"context"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// TestContextCancellation tests that operations respect context cancellation
func TestContextCancellation(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)
	csp := NewMockCloudStorageProvider(am)

	// Register and login
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	am.Register(creds)

	// Set a delay to simulate slow operation
	csp.SetUploadDelay(2 * time.Second)

	// Create a context that will be canceled
	ctx, cancel := context.WithCancel(context.Background())

	// Start upload in goroutine
	errChan := make(chan error)
	go func() {
		err := csp.Upload(ctx, types.PetID("test-pet"), []byte("data"))
		errChan <- err
	}()

	// Cancel the context immediately
	cancel()

	// Wait for the upload to complete
	err := <-errChan
	if err == nil {
		t.Error("Expected error from canceled context")
	}
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}

// TestContextTimeout tests that operations respect context timeouts
func TestContextTimeout(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)
	csp := NewMockCloudStorageProvider(am)

	// Register and login
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	am.Register(creds)

	// Set a delay longer than timeout
	csp.SetUploadDelay(2 * time.Second)

	// Create a context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Attempt upload
	err := csp.Upload(ctx, types.PetID("test-pet"), []byte("data"))
	if err == nil {
		t.Error("Expected error from timeout")
	}
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded error, got %v", err)
	}
}

// TestContextCancellationDownload tests download cancellation
func TestContextCancellationDownload(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)
	csp := NewMockCloudStorageProvider(am)

	// Register and login
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	am.Register(creds)

	// Upload a pet first
	ctx := context.Background()
	petID := types.PetID("test-pet")
	csp.Upload(ctx, petID, []byte("data"))

	// Set download delay
	csp.SetDownloadDelay(2 * time.Second)

	// Create a context that will be canceled
	ctxCancel, cancel := context.WithCancel(context.Background())

	// Start download in goroutine
	errChan := make(chan error)
	go func() {
		_, err := csp.Download(ctxCancel, petID)
		errChan <- err
	}()

	// Cancel the context immediately
	cancel()

	// Wait for the download to complete
	err := <-errChan
	if err == nil {
		t.Error("Expected error from canceled context")
	}
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}

// TestContextPropagation tests that context is properly propagated through methods
func TestContextPropagation(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)
	csp := NewMockCloudStorageProvider(am)

	// Register and login
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	am.Register(creds)

	// Test all methods with canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Upload should fail
	err := csp.Upload(ctx, types.PetID("test"), []byte("data"))
	if err != context.Canceled {
		t.Errorf("Upload: expected context.Canceled, got %v", err)
	}

	// Download should fail
	_, err = csp.Download(ctx, types.PetID("test"))
	if err != context.Canceled {
		t.Errorf("Download: expected context.Canceled, got %v", err)
	}

	// Delete should fail
	err = csp.Delete(ctx, types.PetID("test"))
	if err != context.Canceled {
		t.Errorf("Delete: expected context.Canceled, got %v", err)
	}

	// List should fail
	_, err = csp.List(ctx)
	if err != context.Canceled {
		t.Errorf("List: expected context.Canceled, got %v", err)
	}

	// GetLastModified should fail
	_, err = csp.GetLastModified(ctx, types.PetID("test"))
	if err != context.Canceled {
		t.Errorf("GetLastModified: expected context.Canceled, got %v", err)
	}

	// IsConnected should return false
	if csp.IsConnected(ctx) {
		t.Error("IsConnected: expected false for canceled context")
	}
}

// TestContextDeadline tests operations with various deadline scenarios
func TestContextDeadline(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)
	csp := NewMockCloudStorageProvider(am)

	// Register and login
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	am.Register(creds)

	t.Run("sufficient time", func(t *testing.T) {
		// Set no delay
		csp.SetUploadDelay(0)

		// Create context with generous timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Upload should succeed
		err := csp.Upload(ctx, types.PetID("test-pet"), []byte("data"))
		if err != nil {
			t.Errorf("Expected successful upload with sufficient time, got %v", err)
		}
	})

	t.Run("insufficient time", func(t *testing.T) {
		// Set long delay
		csp.SetUploadDelay(1 * time.Second)

		// Create context with very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		// Upload should fail
		err := csp.Upload(ctx, types.PetID("test-pet-2"), []byte("data"))
		if err != context.DeadlineExceeded {
			t.Errorf("Expected context.DeadlineExceeded, got %v", err)
		}
	})
}
