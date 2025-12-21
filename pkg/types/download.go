package types

type DownloadConfig struct {
	Level          string
	Output         string
	ConflictPolicy string
}

type DownloadResult struct {
	Err  error            `json:"error,omitempty" description:"Error encountered during download, if any"`
	Data *DownloadedMusic `json:"data,omitempty" description:"Downloaded music information"`
}

type MusicDownloadResults struct {
	results []*DownloadResult
}

func NewMusicDownloadResults(size int) *MusicDownloadResults {
	return &MusicDownloadResults{
		results: make([]*DownloadResult, 0, size),
	}
}
func (d *MusicDownloadResults) Add(result *DownloadResult) {
	if result == nil {
		return
	}
	d.results = append(d.results, result)
}

func (d *MusicDownloadResults) Total() int {
	return len(d.results)
}

func (d *MusicDownloadResults) SuccessCount() int {
	var count int
	for _, r := range d.results {
		if r.Err == nil {
			count++
		}
	}
	return count
}

func (d *MusicDownloadResults) FailedCount() int {
	return d.Total() - d.SuccessCount()
}

func (d *MusicDownloadResults) Results() []*DownloadResult {
	return d.results
}
