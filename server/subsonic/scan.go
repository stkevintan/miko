package subsonic

import (
	"context"
	"net/http"

	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/di"
	"github.com/stkevintan/miko/pkg/scanner"
)

func (s *Subsonic) handleGetScanStatus(w http.ResponseWriter, r *http.Request) {
	sc := di.MustInvoke[*scanner.Scanner](r.Context())
	resp := models.NewResponse(models.ResponseStatusOK)
	resp.ScanStatus = &models.ScanStatus{
		Scanning: sc.IsScanning(),
		Count:    sc.ScanCount(),
	}
	s.sendResponse(w, r, resp)
}

func (s *Subsonic) handleStartScan(w http.ResponseWriter, r *http.Request) {
	// default to incremental scan
	incremental := !isPositive(r.URL.Query().Get("full"))
	sc := di.MustInvoke[*scanner.Scanner](r.Context())
	// r.Context() may destroy on client disconnect, so create a new background context
	ctx := di.Inherit(context.Background(), r.Context())
	go sc.Scan(ctx, incremental)
	resp := models.NewResponse(models.ResponseStatusOK)
	resp.ScanStatus = &models.ScanStatus{
		Scanning: true,
	}
	s.sendResponse(w, r, resp)
}

func isPositive(s string) bool {
	return s == "1" || s == "true" || s == "yes"
}
