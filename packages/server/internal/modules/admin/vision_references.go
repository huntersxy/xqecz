package admin

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/huntersxy/xqecz/server/internal/web"
)

func (h *Handler) uploadVisionReference(c *gin.Context) {
	if !h.hasVision(c) {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 21<<20)
	reader, err := c.Request.MultipartReader()
	if err != nil {
		web.SoftFail(c, 400, "需要 multipart 图片上传")
		return
	}
	var file *os.File
	defer func() {
		if file != nil {
			file.Close()
			os.Remove(file.Name())
		}
	}()
	name := c.Param("name")
	fallback := ""
	for {
		part, e := reader.NextPart()
		if e == io.EOF {
			break
		}
		if e != nil {
			web.SoftFail(c, 400, "上传体过大或读取失败")
			return
		}
		if part.FormName() == "file" && part.FileName() != "" {
			if file != nil {
				part.Close()
				web.SoftFail(c, 400, "一次只能上传一张参照图")
				return
			}
			fallback = strings.TrimSuffix(filepath.Base(part.FileName()), filepath.Ext(part.FileName()))
			file, e = os.CreateTemp("", "xqecz-reference-*")
			if e != nil {
				part.Close()
				web.SoftFail(c, 500, "无法暂存参照图")
				return
			}
			n, e := io.Copy(file, io.LimitReader(part, (20<<20)+1))
			part.Close()
			if e != nil || n > 20<<20 {
				web.SoftFail(c, 400, "参照图不能超过 20MB")
				return
			}
		} else {
			b, e := io.ReadAll(io.LimitReader(part, 513))
			part.Close()
			if e != nil || len(b) > 512 {
				web.SoftFail(c, 400, "上传字段过长")
				return
			}
			if part.FormName() == "name" && c.Param("name") == "" {
				name = strings.TrimSpace(string(b))
			}
		}
	}
	if file == nil {
		web.SoftFail(c, 400, "请选择参照图片")
		return
	}
	if name == "" {
		name = fallback
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		web.SoftFail(c, 500, "参照图读取失败")
		return
	}
	res, err := h.deps.Vision.PutReference(c.Request.Context(), name, file, c.Param("name") != "")
	if err != nil {
		web.SoftFail(c, 400, err.Error())
		return
	}
	web.OK(c, res, "参照图已保存并压缩")
}

func (h *Handler) visionReferenceImage(c *gin.Context) {
	if !h.hasVision(c) {
		return
	}
	b, err := h.deps.Vision.ReferenceImage(c.Request.Context(), c.Param("name"))
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, "image/jpeg", b)
}
func bindReferenceJSON(c *gin.Context, v any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16<<10)
	if c.ShouldBindJSON(v) != nil {
		web.SoftFail(c, 400, "请求参数格式错误")
		return false
	}
	return true
}
func (h *Handler) renameVisionReference(c *gin.Context) {
	if !h.hasVision(c) {
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if !bindReferenceJSON(c, &req) {
		return
	}
	res, err := h.deps.Vision.RenameReference(c.Request.Context(), c.Param("name"), req.Name)
	if err != nil {
		web.SoftFail(c, 400, err.Error())
		return
	}
	web.OK(c, res, "角色名已更新")
}
func (h *Handler) deleteVisionReference(c *gin.Context) {
	if !h.hasVision(c) {
		return
	}
	if err := h.deps.Vision.DeleteReference(c.Request.Context(), c.Param("name")); err != nil {
		web.SoftFail(c, 400, err.Error())
		return
	}
	web.OK(c, nil, "已移出参照库，原图保留在后台备份目录")
}
func (h *Handler) saveVisionReferenceDescription(c *gin.Context) {
	if !h.hasVision(c) {
		return
	}
	var req struct {
		Description string `json:"description"`
		Version     string `json:"version"`
	}
	if !bindReferenceJSON(c, &req) {
		return
	}
	res, err := h.deps.Vision.SaveReferenceDescription(c.Request.Context(), c.Param("name"), req.Description, req.Version)
	if err != nil {
		web.SoftFail(c, 400, err.Error())
		return
	}
	web.OK(c, res, "文字描述已保存")
}
func (h *Handler) describeVisionReference(c *gin.Context) {
	if !h.hasVision(c) {
		return
	}
	var req struct {
		Version string `json:"version"`
	}
	if !bindReferenceJSON(c, &req) {
		return
	}
	if req.Version == "" {
		web.SoftFail(c, 400, "请刷新参照库后重试")
		return
	}
	res, err := h.deps.Vision.DescribeReference(c.Request.Context(), c.Param("name"), req.Version)
	if err != nil {
		web.SoftFail(c, 400, err.Error())
		return
	}
	web.OK(c, res, "参照图已转述为文字特征")
}
