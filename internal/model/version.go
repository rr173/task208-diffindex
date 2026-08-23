package model

import (
	"fmt"
	"time"
)

// VersionStatus 索引版本的生命周期状态。
type VersionStatus string

const (
	// VersionDraft 草稿版本，尚未发布。
	VersionDraft VersionStatus = "draft"
	// VersionPublished 已发布的当前索引版本。
	VersionPublished VersionStatus = "published"
	// VersionSuperseded 被更新版本替代。
	VersionSuperseded VersionStatus = "superseded"
)

// IndexVersion 一次索引发布的不可变快照：绑定确认晶格与排除峰清单。
type IndexVersion struct {
	ID            string        `json:"id"`
	BatchID       string        `json:"batch_id"`
	Number        int           `json:"number"`
	LatticeID     string        `json:"lattice_id"`
	Status        VersionStatus `json:"status"`
	ExcludedPeaks []string      `json:"excluded_peaks"`
	PeakCount     int           `json:"peak_count"`
	MeanResidual  float64       `json:"mean_residual"`
	CreatedAt     time.Time     `json:"created_at"`
	PublishedAt   *time.Time    `json:"published_at,omitempty"`
}

// NewIndexVersion 构造草稿版本。排除清单会拷贝一份，避免外部修改污染快照。
func NewIndexVersion(id, batchID, latticeID string, number, peakCount int, meanResidual float64, excluded []string) (*IndexVersion, error) {
	if id == "" || batchID == "" || latticeID == "" {
		return nil, InvalidInputf("version id, batch id and lattice id must not be empty")
	}
	if number <= 0 {
		return nil, InvalidInputf("version number must be positive, got %d", number)
	}
	excl := make([]string, len(excluded))
	copy(excl, excluded)
	return &IndexVersion{
		ID: id, BatchID: batchID, LatticeID: latticeID,
		Number: number, Status: VersionDraft,
		ExcludedPeaks: excl, PeakCount: peakCount, MeanResidual: meanResidual,
		CreatedAt: time.Now().UTC(),
	}, nil
}

// Publish 发布版本。
func (v *IndexVersion) Publish() error {
	if v.Status == VersionSuperseded {
		return fmt.Errorf("%w: superseded version cannot be republished", ErrInvalidState)
	}
	if v.Status == VersionPublished {
		return nil
	}
	now := time.Now().UTC()
	v.Status = VersionPublished
	v.PublishedAt = &now
	return nil
}

// Supersede 将版本标记为被替代。
func (v *IndexVersion) Supersede() error {
	if v.Status != VersionPublished {
		return fmt.Errorf("%w: only published version can be superseded", ErrInvalidState)
	}
	v.Status = VersionSuperseded
	return nil
}
