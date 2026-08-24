package bloodhound

import (
	"context"
	"fmt"
	"mime"
	"path/filepath"
	"strings"
	"time"

	bhservices "github.com/Gubarz/bloodhound-sdk-go/services"
)

// Event types published on the app bus for collection upload jobs.
const (
	EventIngestStarted   = "bloodhound.ingest.job.started"
	EventIngestProgress  = "bloodhound.ingest.job.progress"
	EventIngestCompleted = "bloodhound.ingest.job.completed"
	EventIngestFailed    = "bloodhound.ingest.job.failed"
)

// IngestJobDTO is the wire shape for collection upload jobs.
type IngestJobDTO struct {
	ID        int64           `json:"id"`
	Status    string          `json:"status"`
	Message   string          `json:"message,omitempty"`
	CreatedAt int64           `json:"createdAt"` // unix milli
	Total     int             `json:"totalFiles"`
	Failed    int             `json:"failedFiles"`
	Files     []IngestFileDTO `json:"files,omitempty"`
}

// IngestFileDTO is one file's processing result.
type IngestFileDTO struct {
	Name   string   `json:"name"`
	Errors []string `json:"errors,omitempty"`
}

// jobStatusLabel maps the SDK's job status enum (-1..8) to stable strings
// for the UI.
func jobStatusLabel(status int) string {
	switch status {
	case -1:
		return "invalid"
	case 0:
		return "ready"
	case 1:
		return "running"
	case 2:
		return "complete"
	case 3:
		return "canceled"
	case 4:
		return "timed_out"
	case 5:
		return "failed"
	case 6:
		return "ingesting"
	case 7:
		return "analyzing"
	case 8:
		return "partially_complete"
	default:
		return "unknown"
	}
}

func jobTerminal(status int) bool {
	return status == 2 || status == 3 || status == 4 || status == 5 || status == 8
}

func jobFailed(status int) bool {
	return status == 3 || status == 4 || status == 5
}

func jobDTOFromSDK(job *bhservices.FileUploadJob, files []IngestFileDTO) IngestJobDTO {
	dto := IngestJobDTO{Files: files}
	if job == nil {
		return dto
	}
	if job.Id != nil {
		dto.ID = *job.Id
	}
	if job.Status != nil {
		dto.Status = jobStatusLabel(int(*job.Status))
	}
	if job.StatusMessage != nil {
		dto.Message = *job.StatusMessage
	}
	if job.CreatedAt != nil {
		dto.CreatedAt = job.CreatedAt.UnixMilli()
	}
	if job.TotalFiles != nil {
		dto.Total = *job.TotalFiles
	}
	if job.FailedFiles != nil {
		dto.Failed = *job.FailedFiles
	}
	return dto
}

func ingestFileDTOs(tasks []bhservices.FileUploadJobCompletedTasks) []IngestFileDTO {
	out := make([]IngestFileDTO, 0, len(tasks))
	for _, task := range tasks {
		f := IngestFileDTO{Name: deref(task.FileName)}
		if task.Errors != nil {
			f.Errors = append([]string{}, *task.Errors...)
		}
		out = append(out, f)
	}
	return out
}

// IngestBytes uploads collection data (e.g. a SharpHound zip) to BloodHound
// and returns the fresh job record. Publishes bloodhound.ingest.job.started.
func (s *Service) IngestBytes(ctx context.Context, name, contentType string, data []byte) (IngestJobDTO, error) {
	client, err := s.snapshot()
	if err != nil {
		return IngestJobDTO{}, err
	}
	if contentType == "" {
		contentType = detectContentType(name)
	}
	job, err := client.Community().CollectionUploads().UploadFiles(ctx, []bhservices.UploadFile{
		{Path: name, ContentType: contentType, Data: data},
	})
	if err != nil {
		return IngestJobDTO{}, err
	}
	dto := jobDTOFromSDK(job, []IngestFileDTO{{Name: name}})
	s.publish(EventIngestStarted, dto)
	return dto, nil
}

// IngestLocalFile reads a local zip/json and uploads it.
func (s *Service) IngestLocalFile(ctx context.Context, path string) (IngestJobDTO, error) {
	client, err := s.snapshot()
	if err != nil {
		return IngestJobDTO{}, err
	}
	job, err := client.Community().CollectionUploads().UploadFilePaths(ctx, path)
	if err != nil {
		return IngestJobDTO{}, err
	}
	name := filepath.Base(path)
	dto := jobDTOFromSDK(job, []IngestFileDTO{{Name: name}})
	s.publish(EventIngestStarted, dto)
	return dto, nil
}

// IngestJobs lists upload jobs (no per-file detail).
func (s *Service) IngestJobs(ctx context.Context) ([]IngestJobDTO, error) {
	client, err := s.snapshot()
	if err != nil {
		return nil, err
	}
	jobs, err := client.Community().CollectionUploads().FileUploadJobs(ctx, nil)
	if err != nil {
		return nil, err
	}
	out := make([]IngestJobDTO, 0, len(jobs))
	for i := range jobs {
		out = append(out, jobDTOFromSDK(&jobs[i], nil))
	}
	return out, nil
}

// IngestJob returns one job with per-file rows from CompletedTasks.
func (s *Service) IngestJob(ctx context.Context, id int64) (IngestJobDTO, error) {
	client, err := s.snapshot()
	if err != nil {
		return IngestJobDTO{}, err
	}
	job, err := client.Community().CollectionUploads().FileUploadJob(ctx, id)
	if err != nil {
		return IngestJobDTO{}, err
	}
	tasks, err := client.Community().CollectionUploads().CompletedTasks(ctx, id)
	if err != nil {
		return IngestJobDTO{}, err
	}
	return jobDTOFromSDK(job, ingestFileDTOs(tasks)), nil
}

// WatchIngestJob polls a job until it reaches a terminal status, publishing
// progress/completed/failed events. On success the correlation cache is
// invalidated so enrichment reflects the freshly ingested data.
func (s *Service) WatchIngestJob(ctx context.Context, id int64, every time.Duration) error {
	if every <= 0 {
		every = 2 * time.Second
	}
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	var lastStatus string
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			job, err := s.IngestJob(ctx, id)
			if err != nil {
				return err
			}
			if job.Status != lastStatus {
				lastStatus = job.Status
				s.publish(EventIngestProgress, job)
			}
			status := ingestStatusInt(job.Status)
			if !jobTerminal(status) {
				continue
			}
			if jobFailed(status) {
				s.publish(EventIngestFailed, job)
				return fmt.Errorf("bloodhound: ingest job %d %s: %s", id, job.Status, job.Message)
			}
			s.InvalidateCorrelation()
			s.publish(EventIngestCompleted, job)
			return nil
		}
	}
}

// ingestStatusInt maps a DTO status label back to the SDK enum int, used
// only inside WatchIngestJob for terminal checks.
func ingestStatusInt(label string) int {
	switch label {
	case "invalid":
		return -1
	case "ready":
		return 0
	case "running":
		return 1
	case "complete":
		return 2
	case "canceled":
		return 3
	case "timed_out":
		return 4
	case "failed":
		return 5
	case "ingesting":
		return 6
	case "analyzing":
		return 7
	case "partially_complete":
		return 8
	default:
		return -1
	}
}

func detectContentType(name string) string {
	if ct := mime.TypeByExtension(strings.ToLower(filepath.Ext(name))); ct != "" {
		return ct
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".json":
		return "application/json"
	case ".zip":
		return "application/zip"
	default:
		return "application/octet-stream"
	}
}
