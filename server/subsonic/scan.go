package subsonic

import (
	"context"
	"net/http"

	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/di"
	"github.com/stkevintan/miko/pkg/scanner"
	"gorm.io/gorm"
)

func (s *Subsonic) handleGetScanStatus(w http.ResponseWriter, r *http.Request) {
	sc := di.MustInvoke[*scanner.Scanner](r.Context())
	db := di.MustInvoke[*gorm.DB](r.Context())

	var count int64
	db.Model(&models.Child{}).Where("is_dir = ?", false).Count(&count)

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.ScanStatus = &models.ScanStatus{
		Scanning: sc.IsScanning(),
		Count:    count,
	}
	s.sendResponse(w, r, resp)
}

func (s *Subsonic) handleStartScan(w http.ResponseWriter, r *http.Request) {
	incremental := isPositive(r.URL.Query().Get("inc"))
	sc := di.MustInvoke[*scanner.Scanner](r.Context())
	db := di.MustInvoke[*gorm.DB](r.Context())

	var count int64
	db.Model(&models.Child{}).Where("is_dir = ?", false).Count(&count)

	// r.Context() may destroy on client disconnect, so create a new background context
	ctx := di.Inherit(context.Background(), r.Context())
	go sc.Scan(ctx, incremental)
	resp := models.NewResponse(models.ResponseStatusOK)
	resp.ScanStatus = &models.ScanStatus{
		Scanning: true,
		Count:    count,
	}
	s.sendResponse(w, r, resp)
}

func isPositive(s string) bool {
	return s == "1" || s == "true" || s == "yes"
}
