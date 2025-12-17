package service

import (
	"testing"
)

func TestParse(t *testing.T) {
	service := &Service{}
	
	tests := []struct {
		name        string
		input       string
		expectedKind string
		expectedID   int64
		expectError bool
	}{
		{
			name:        "Direct song ID",
			input:       "2161154646",
			expectedKind: "song",
			expectedID:   2161154646,
			expectError: false,
		},
		{
			name:        "Song URL",
			input:       "https://music.163.com/song?id=2161154646",
			expectedKind: "song",
			expectedID:   2161154646,
			expectError: false,
		},
		{
			name:        "Playlist URL with hash fragment",
			input:       "https://music.163.com/#/playlist?id=3160902515",
			expectedKind: "playlist",
			expectedID:   3160902515,
			expectError: false,
		},
		{
			name:        "Album URL",
			input:       "https://music.163.com/album?id=123456",
			expectedKind: "album",
			expectedID:   123456,
			expectError: false,
		},
		{
			name:        "Artist URL",
			input:       "https://music.163.com/artist?id=789012",
			expectedKind: "artist",
			expectedID:   789012,
			expectError: false,
		},
		{
			name:        "Invalid URL - wrong domain",
			input:       "https://example.com/song?id=123",
			expectError: true,
		},
		{
			name:        "Invalid input",
			input:       "not-a-number-or-url",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kind, id, err := service.Parse(tt.input)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			
			if kind != tt.expectedKind {
				t.Errorf("Expected kind %s, got %s", tt.expectedKind, kind)
			}
			
			if id != tt.expectedID {
				t.Errorf("Expected ID %d, got %d", tt.expectedID, id)
			}
		})
	}
}