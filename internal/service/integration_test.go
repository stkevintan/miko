package service

import (
	"testing"
)

func TestServiceIntegration(t *testing.T) {
	// Test that the basic service structure and methods are available

	// Test Parse method
	service := &Service{}

	// Test URL parsing
	t.Run("Parse URL with fragment", func(t *testing.T) {
		kind, id, err := service.Parse("https://music.163.com/#/playlist?id=3160902515")
		if err != nil {
			t.Errorf("Parse failed: %v", err)
		}
		if kind != "playlist" {
			t.Errorf("Expected kind 'playlist', got '%s'", kind)
		}
		if id != 3160902515 {
			t.Errorf("Expected id 3160902515, got %d", id)
		}
	})

	// Test song ID parsing
	t.Run("Parse direct song ID", func(t *testing.T) {
		kind, id, err := service.Parse("2161154646")
		if err != nil {
			t.Errorf("Parse failed: %v", err)
		}
		if kind != "song" {
			t.Errorf("Expected kind 'song', got '%s'", kind)
		}
		if id != 2161154646 {
			t.Errorf("Expected id 2161154646, got %d", id)
		}
	})

	// Test invalid input
	t.Run("Parse invalid input", func(t *testing.T) {
		_, _, err := service.Parse("invalid-input")
		if err == nil {
			t.Error("Expected error for invalid input")
		}
	})
}
