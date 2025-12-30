package subsonic

import (
	"net/http"

	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/bookmarks"
	"github.com/stkevintan/miko/pkg/di"
)

func (s *Subsonic) handleGetBookmarks(w http.ResponseWriter, r *http.Request) {
	username := string(di.MustInvoke[models.Username](r.Context()))
	bm := di.MustInvoke[*bookmarks.Manager](r.Context())

	bookmarks, err := bm.GetBookmarks(username)
	if err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(0, err.Error()))
		return
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.Bookmarks = &models.Bookmarks{
		Bookmark: bookmarks,
	}
	s.sendResponse(w, r, resp)
}

func (s *Subsonic) handleCreateBookmark(w http.ResponseWriter, r *http.Request) {
	username := string(di.MustInvoke[models.Username](r.Context()))
	bm := di.MustInvoke[*bookmarks.Manager](r.Context())

	id := r.URL.Query().Get("id")
	position := getQueryIntOrDefault(r, "position", 0)
	comment := r.URL.Query().Get("comment")

	if err := bm.CreateBookmark(username, id, int64(position), comment); err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(0, err.Error()))
		return
	}

	s.sendResponse(w, r, models.NewResponse(models.ResponseStatusOK))
}

func (s *Subsonic) handleDeleteBookmark(w http.ResponseWriter, r *http.Request) {
	username := string(di.MustInvoke[models.Username](r.Context()))
	bm := di.MustInvoke[*bookmarks.Manager](r.Context())

	id := r.URL.Query().Get("id")

	if err := bm.DeleteBookmark(username, id); err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(0, err.Error()))
		return
	}

	s.sendResponse(w, r, models.NewResponse(models.ResponseStatusOK))
}

func (s *Subsonic) handleGetPlayQueue(w http.ResponseWriter, r *http.Request) {
	username := string(di.MustInvoke[models.Username](r.Context()))
	bm := di.MustInvoke[*bookmarks.Manager](r.Context())

	queue, err := bm.GetPlayQueue(username)
	if err != nil {
		// If not found, return empty queue instead of error
		resp := models.NewResponse(models.ResponseStatusOK)
		resp.PlayQueue = &models.PlayQueue{
			Username: username,
			Entry:    []models.Child{},
		}
		s.sendResponse(w, r, resp)
		return
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.PlayQueue = queue
	s.sendResponse(w, r, resp)
}

func (s *Subsonic) handleSavePlayQueue(w http.ResponseWriter, r *http.Request) {
	username := string(di.MustInvoke[models.Username](r.Context()))
	bm := di.MustInvoke[*bookmarks.Manager](r.Context())

	current := r.URL.Query().Get("current")
	position := getQueryIntOrDefault(r, "position", 0)
	songIDs := r.URL.Query()["id"]
	clientName := r.URL.Query().Get("c")

	if err := bm.SavePlayQueue(username, current, int64(position), songIDs, clientName); err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(0, err.Error()))
		return
	}

	s.sendResponse(w, r, models.NewResponse(models.ResponseStatusOK))
}
