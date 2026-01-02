package subsonic

import (
	"net/http"

	"github.com/stkevintan/miko/config"
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
	query := r.URL.Query()
	var incremental bool
	if query.Has("inc") {
		incremental = isPositive(query.Get("inc"))
	} else {
		cfg := di.MustInvoke[*config.Config](r.Context())
		incremental = cfg.Subsonic.ScanMode != "full"
	}

	sc := di.MustInvoke[*scanner.Scanner](r.Context())
	db := di.MustInvoke[*gorm.DB](r.Context())

	var count int64
	db.Model(&models.Child{}).Where("is_dir = ?", false).Count(&count)

	// Use app context so scan survives request disconnect but stops on app shutdown
	go sc.ScanAll(s.ctx, incremental)
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
