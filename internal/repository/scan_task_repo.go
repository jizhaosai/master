package repository

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"

	"mdns-mapper/internal/model"
)

// ScanTaskRepo 扫描任务仓储。
type ScanTaskRepo struct {
	db *gorm.DB
}

// NewScanTaskRepo 创建扫描任务仓储。
func NewScanTaskRepo(db *gorm.DB) *ScanTaskRepo {
	return &ScanTaskRepo{db: db}
}

// Create 创建扫描任务记录。
func (r *ScanTaskRepo) Create(task *model.ScanTask) error {
	return r.db.Create(task).Error
}

// UpdateStatus 更新任务状态。
func (r *ScanTaskRepo) UpdateStatus(id uint64, status string, errMsg string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if errMsg != "" {
		updates["err_msg"] = errMsg
	}
	if status == "running" {
		now := time.Now()
		updates["started_at"] = &now
	}
	if status == "done" || status == "failed" {
		now := time.Now()
		updates["finished_at"] = &now
	}
	return r.db.Model(&model.ScanTask{}).Where("id = ?", id).Updates(updates).Error
}

// UpdateAssetCount 更新资产计数。
func (r *ScanTaskRepo) UpdateAssetCount(id uint64, count int) error {
	return r.db.Model(&model.ScanTask{}).Where("id = ?", id).Update("asset_count", count).Error
}

// GetByID 根据 ID 获取任务（含资产和服务）。
func (r *ScanTaskRepo) GetByID(id uint64) (*model.ScanTask, error) {
	var task model.ScanTask
	err := r.db.Preload("Assets.Services").First(&task, id).Error
	if err != nil {
		return nil, err
	}
	// 反序列化 TxtRecordsJSON → TxtRecords
	for i := range task.Assets {
		for j := range task.Assets[i].Services {
			svc := &task.Assets[i].Services[j]
			if svc.TxtRecordsJSON != "" {
				_ = json.Unmarshal([]byte(svc.TxtRecordsJSON), &svc.TxtRecords)
			}
		}
	}
	return &task, nil
}

// List 获取任务列表（分页）。
func (r *ScanTaskRepo) List(page, pageSize int) ([]model.ScanTask, int64, error) {
	var total int64
	r.db.Model(&model.ScanTask{}).Count(&total)

	var tasks []model.ScanTask
	offset := (page - 1) * pageSize
	err := r.db.Order("id DESC").Offset(offset).Limit(pageSize).Find(&tasks).Error
	return tasks, total, err
}

// SaveAssets 保存扫描到的资产列表（关联到任务）。
func (r *ScanTaskRepo) SaveAssets(taskID uint64, assets []model.Asset) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for i := range assets {
			assets[i].TaskID = taskID
			if err := tx.Create(&assets[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
