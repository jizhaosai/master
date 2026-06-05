package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"mdns-mapper/internal/model"
	"mdns-mapper/internal/output"
	"mdns-mapper/internal/repository"
	"mdns-mapper/internal/service"
	"mdns-mapper/pkg/netutil"
)

// Handler API 处理器。
type Handler struct {
	scanSvc  *service.ScanService
	taskRepo *repository.ScanTaskRepo
	log      *zap.Logger
}

// NewHandler 创建处理器。
func NewHandler(scanSvc *service.ScanService, taskRepo *repository.ScanTaskRepo, log *zap.Logger) *Handler {
	return &Handler{scanSvc: scanSvc, taskRepo: taskRepo, log: log}
}

// 统一响应结构
type response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

func ok(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, response{Code: 0, Msg: "ok", Data: data})
}

func fail(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusOK, response{Code: code, Msg: msg})
}

// CreateScanRequest 发起扫描请求体。
type CreateScanRequest struct {
	CIDR       string `json:"cidr" binding:"required"`
	PortRange  string `json:"port_range" binding:"required"`
	Interface  string `json:"interface"`
	ListenSecs int    `json:"listen_secs"`
}

// CreateScan 发起扫描任务。
func (h *Handler) CreateScan(c *gin.Context) {
	var req CreateScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误: "+err.Error())
		return
	}

	listenSecs := req.ListenSecs
	if listenSecs <= 0 {
		listenSecs = 5
	}

	// 创建任务记录
	task := &model.ScanTask{
		CIDR:      req.CIDR,
		PortRange: req.PortRange,
		Interface: req.Interface,
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	if err := h.taskRepo.Create(task); err != nil {
		fail(c, 500, "创建任务失败: "+err.Error())
		return
	}

	// 异步执行扫描
	go h.executeScan(task.ID, req, listenSecs)

	ok(c, gin.H{"task_id": task.ID, "status": "pending"})
}

// executeScan 异步执行扫描并保存结果。
func (h *Handler) executeScan(taskID uint64, req CreateScanRequest, listenSecs int) {
	// 更新为 running
	_ = h.taskRepo.UpdateStatus(taskID, "running", "")

	input := service.ScanInput{
		CIDR:        req.CIDR,
		Ports:       req.PortRange,
		Interface:   req.Interface,
		ListenDur:   time.Duration(listenSecs) * time.Second,
		Retries:     3,
		Unicast:     true,
		Concurrency: 256,
	}

	ctx, cancel := contextWithTimeout(time.Duration(listenSecs+30) * time.Second)
	defer cancel()
	result, err := h.scanSvc.Run(ctx, input)
	if err != nil {
		h.log.Error("扫描失败", zap.Uint64("task_id", taskID), zap.Error(err))
		_ = h.taskRepo.UpdateStatus(taskID, "failed", err.Error())
		return
	}

	// 保存资产
	if err := h.taskRepo.SaveAssets(taskID, result.Assets); err != nil {
		h.log.Error("保存资产失败", zap.Uint64("task_id", taskID), zap.Error(err))
		_ = h.taskRepo.UpdateStatus(taskID, "failed", "保存资产失败: "+err.Error())
		return
	}

	_ = h.taskRepo.UpdateAssetCount(taskID, result.Total)
	_ = h.taskRepo.UpdateStatus(taskID, "done", "")
	h.log.Info("扫描完成", zap.Uint64("task_id", taskID), zap.Int("assets", result.Total))
}

// ListScans 获取任务列表。
func (h *Handler) ListScans(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	tasks, total, err := h.taskRepo.List(page, pageSize)
	if err != nil {
		fail(c, 500, "查询失败: "+err.Error())
		return
	}
	ok(c, gin.H{"tasks": tasks, "total": total, "page": page, "page_size": pageSize})
}

// GetScan 获取单个任务详情。
func (h *Handler) GetScan(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, 400, "任务 ID 无效")
		return
	}
	task, err := h.taskRepo.GetByID(id)
	if err != nil {
		fail(c, 404, "任务不存在")
		return
	}
	ok(c, task)
}

// GetScanAssets 获取任务的资产列表。
func (h *Handler) GetScanAssets(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, 400, "任务 ID 无效")
		return
	}
	task, err := h.taskRepo.GetByID(id)
	if err != nil {
		fail(c, 404, "任务不存在")
		return
	}
	ok(c, gin.H{"assets": task.Assets, "total": len(task.Assets)})
}

// ExportScan 导出扫描结果。
func (h *Handler) ExportScan(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, 400, "任务 ID 无效")
		return
	}
	task, err := h.taskRepo.GetByID(id)
	if err != nil {
		fail(c, 404, "任务不存在")
		return
	}

	format := c.DefaultQuery("format", "text")
	switch format {
	case "json":
		jsonStr, _ := output.FormatJSON(task.Assets)
		c.Header("Content-Type", "application/json")
		c.String(http.StatusOK, jsonStr)
	default:
		text := output.FormatText(task.Assets)
		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.String(http.StatusOK, text)
	}
}

// ListInterfaces 列出可用网卡。
func (h *Handler) ListInterfaces(c *gin.Context) {
	ifaces, err := netutil.ListInterfaces()
	if err != nil {
		fail(c, 500, "获取网卡列表失败: "+err.Error())
		return
	}
	ok(c, ifaces)
}
