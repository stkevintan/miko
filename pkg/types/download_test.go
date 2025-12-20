package types

import "testing"

func TestNewMusicDownloadResults_DoesNotPreFillWithNilEntries(t *testing.T) {
	r := NewMusicDownloadResults(3)
	if r.Total() != 0 {
		t.Fatalf("expected Total()=0 for new results, got %d", r.Total())
	}

	r.Add(&DownloadResult{Err: nil})
	r.Add(&DownloadResult{Err: assertErr{}})

	if r.Total() != 2 {
		t.Fatalf("expected Total()=2 after adds, got %d", r.Total())
	}
	if r.SuccessCount() != 1 {
		t.Fatalf("expected SuccessCount()=1, got %d", r.SuccessCount())
	}
	if r.FailedCount() != 1 {
		t.Fatalf("expected FailedCount()=1, got %d", r.FailedCount())
	}
}

func TestMusicDownloadResults_SuccessCount_NilEntriesAreSafe(t *testing.T) {
	r := NewMusicDownloadResults(0)
	r.Add(nil) // should not panic
	r.Add(&DownloadResult{Err: nil})

	if r.SuccessCount() != 1 {
		t.Fatalf("expected SuccessCount()=1, got %d", r.SuccessCount())
	}
}

type assertErr struct{}

func (assertErr) Error() string { return "err" }
