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

func (d *MusicDownloadResults) Total() int64 {
	return int64(len(d.results))
}

func (d *MusicDownloadResults) SuccessCount() int64 {
	var count int64
	for _, r := range d.results {
		if r.Err == nil {
			count++
		}
	}
	return count
}

func (d *MusicDownloadResults) FailedCount() int64 {
	return d.Total() - d.SuccessCount()
}

func (d *MusicDownloadResults) Results() []*DownloadResult {
	return d.results
}
