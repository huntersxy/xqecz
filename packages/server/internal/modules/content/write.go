package content

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/huntersxy/xqecz/server/internal/media"
	"github.com/huntersxy/xqecz/server/internal/store"
	"github.com/huntersxy/xqecz/server/internal/web"
	"gorm.io/gorm"
)

// softFail 返回 HTTP 200 + 业务错误码（与旧实现一致：校验类失败不改变 HTTP 状态）。
func softFail(c *gin.Context, code int, message string) { web.SoftFail(c, code, message) }

// randomName 生成兜底文件名（第三方客户端未按 md5 重命名时使用）。
func randomName() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return strconv.FormatInt(time.Now().UnixMilli(), 10) + "_" + hex.EncodeToString(b)
}

// createInput 是创建内容的统一入参（普通上传与游客快速上传共用）。
type createInput struct {
	Title         string
	Content       string
	FilePath      string
	FileSize      int64
	Tags          []string
	UserID        uint64
	AuditStatus   string
	GuestNickname string
	GuestEmail    string
	AbsPath       string
}

// upload 登录用户上传（Session 或具备 upload 权限的 API 密钥）。
func (h *Handler) upload(c *gin.Context) {
	identity := web.MustIdentity(c)
	form, err := h.parseMultipart(c)
	if err != nil {
		respondUploadError(c, err)
		return
	}

	title := strings.TrimSpace(form.value("title"))
	content := form.value("content")
	if !validTitle(title) {
		softFail(c, 400, "标题长度需在 1 到 200 个字符之间")
		return
	}
	if strings.TrimSpace(content) == "" && form.File == nil {
		softFail(c, 400, "描述正文与媒体文件至少填一项")
		return
	}

	relPath, size, absPath := h.prepareUploadFile(form.File)
	audit := "pending"
	if identity.IsAdmin {
		audit = "approved"
	}

	item, err := h.createContent(c.Request.Context(), createInput{
		Title: title, Content: content, FilePath: relPath, FileSize: size,
		Tags: form.parseTagsField(), UserID: identity.UID, AuditStatus: audit, AbsPath: absPath,
	})
	if err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	web.OK(c, item, "上传成功")
}

// quickUpload 游客快速上传（免登录）：邮箱 + 昵称标识上传者，落库 user_id=0。
func (h *Handler) quickUpload(c *gin.Context) {
	ctx := c.Request.Context()
	form, err := h.parseMultipart(c)
	if err != nil {
		respondUploadError(c, err)
		return
	}

	title := strings.TrimSpace(form.value("title"))
	content := form.value("content")
	if !validTitle(title) {
		softFail(c, 400, "标题长度需在 1 到 200 个字符之间")
		return
	}
	if strings.TrimSpace(content) == "" && form.File == nil {
		softFail(c, 400, "描述正文与媒体文件至少填一项")
		return
	}

	// 已登录用户沿用真实身份；未登录用游客信息。
	var user *store.User
	if sid, err := c.Cookie(web.SessionCookie); err == nil && sid != "" {
		if uid, ok := h.deps.Redis.GetSession(ctx, sid); ok {
			var u store.User
			if err := h.deps.DB.WithContext(ctx).First(&u, uid).Error; err == nil {
				user = &u
			}
		}
	}

	// 仅在有文件时按 IP 限频（纯文字描述不占带宽，无需风控）。
	if form.File != nil {
		count := h.deps.Redis.IncrWithTTL(ctx, "quick_upload:ip:"+clientIP(c), time.Hour)
		if count > 20 {
			_ = removeFile(form.File.AbsPath)
			softFail(c, 429, "上传过于频繁，请一小时后再试")
			return
		}
	}

	relPath, size, absPath := h.prepareUploadFile(form.File)

	in := createInput{
		Title: title, Content: content, FilePath: relPath, FileSize: size,
		Tags: form.parseTagsField(), AuditStatus: "pending", AbsPath: absPath,
	}
	if user != nil {
		in.UserID = user.ID
		if user.IsAdmin == 1 {
			in.AuditStatus = "approved"
		}
	} else {
		in.GuestNickname = strings.TrimSpace(form.value("nickname"))
		in.GuestEmail = strings.ToLower(strings.TrimSpace(form.value("email")))
	}

	item, err := h.createContent(ctx, in)
	if err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	web.OK(c, item, "上传成功")
}

// update 编辑内容：仅作者或管理员可改；编辑后回到待审（管理员编辑视为过审）。
func (h *Handler) update(c *gin.Context) {
	ctx := c.Request.Context()
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		web.Fail(c, 404, "内容不存在")
		return
	}
	identity := web.MustIdentity(c)

	form, err := h.parseMultipart(c)
	if err != nil {
		respondUploadError(c, err)
		return
	}

	var row store.Content
	err = h.deps.DB.WithContext(ctx).First(&row, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		web.Fail(c, 404, "内容不存在")
		return
	}
	if err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	if identity.UID != row.UserID && !identity.IsAdmin {
		web.Fail(c, 403, "无权修改该内容")
		return
	}

	updates := map[string]any{}
	if v, ok := form.Fields["title"]; ok && len(v) > 0 {
		updates["title"] = strings.TrimSpace(v[0])
	}
	if v, ok := form.Fields["content"]; ok && len(v) > 0 {
		updates["content"] = v[0]
	}
	if _, ok := form.Fields["tags"]; ok {
		updates["tags"] = marshalTags(form.parseTagsField())
	}

	if form.File != nil {
		relPath, size, absPath := h.prepareUploadFile(form.File)
		if relPath != "" {
			updates["file_path"] = relPath
			updates["file_size"] = size
		}
		audit := "pending"
		if identity.IsAdmin {
			audit = "approved"
		}
		updates["audit_status"] = audit
		if absPath != "" {
			go h.processMedia(id, absPath, MediaTypeForPath(absPath))
		}
	}

	if len(updates) > 0 {
		if err := h.deps.DB.WithContext(ctx).Model(&store.Content{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			web.Fail(c, 500, "服务异常")
			return
		}
		h.invalidate(ctx, id)
	}

	var updated store.Content
	if err := h.deps.DB.WithContext(ctx).First(&updated, id).Error; err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	item := decorate(updated, h.userMapFor(ctx, []store.Content{updated}), h.likeCount(ctx, id), false)
	web.OK(c, item, "更新成功")
}

// remove 删除内容（物理删除）：仅作者或管理员可删。
// 一并清理关联数据（评论及其举报、点赞、收藏），并把不再被引用的媒体文件移入垃圾桶。
func (h *Handler) remove(c *gin.Context) {
	ctx := c.Request.Context()
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		web.Fail(c, 404, "内容不存在")
		return
	}
	identity := web.MustIdentity(c)

	var row store.Content
	err = h.deps.DB.WithContext(ctx).First(&row, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		web.Fail(c, 404, "内容不存在")
		return
	}
	if err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	if identity.UID != row.UserID && !identity.IsAdmin {
		web.Fail(c, 403, "无权删除该内容")
		return
	}

	if err := h.purgeContent(ctx, row); err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	h.removeOrphanMedia(ctx, row.FilePath, row.ThumbPath)
	h.invalidate(ctx, id)
	web.OK(c, nil, "已删除")
}

// purgeContent 在一个事务里物理删除内容及其关联行。
// 回复的 parent_id 置空而非连带删除，保留他人回复内容、避免悬空引用。
func (h *Handler) purgeContent(ctx context.Context, row store.Content) error {
	return h.deps.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var commentIDs []uint64
		if err := tx.Model(&store.Comment{}).Where("content_id = ?", row.ID).
			Pluck("id", &commentIDs).Error; err != nil {
			return err
		}
		if len(commentIDs) > 0 {
			if err := tx.Where("comment_id IN ?", commentIDs).Delete(&store.CommentReport{}).Error; err != nil {
				return err
			}
			if err := tx.Model(&store.Comment{}).Where("parent_id IN ?", commentIDs).
				Update("parent_id", nil).Error; err != nil {
				return err
			}
			if err := tx.Where("content_id = ?", row.ID).Delete(&store.Comment{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("content_id = ?", row.ID).Delete(&store.ContentLike{}).Error; err != nil {
			return err
		}
		if err := tx.Where("content_id = ?", row.ID).Delete(&store.ContentFavorite{}).Error; err != nil {
			return err
		}
		return tx.Delete(&store.Content{}, row.ID).Error
	})
}

// removeOrphanMedia 把媒体文件移入垃圾桶目录（保留而非删除）。
// 同一文件可能被多条内容共用（历史上存在重复上传），故必须先查引用计数。
func (h *Handler) removeOrphanMedia(ctx context.Context, paths ...*string) {
	db := h.deps.DB.WithContext(ctx)
	for _, p := range paths {
		if p == nil || *p == "" {
			continue
		}
		col := "file_path"
		if strings.HasPrefix(*p, "thumbs/") {
			col = "thumb_path"
		}
		var n int64
		if err := db.Model(&store.Content{}).Where(col+" = ?", *p).Count(&n).Error; err != nil || n > 0 {
			continue
		}
		if err := media.MoveToBin(h.absMediaPath(*p), h.deps.Cfg.BinDir); err != nil {
			slog.Warn("媒体文件入桶失败", "path", *p, "err", err)
		}
	}
}

// claim 提交认领申请。
func (h *Handler) claim(c *gin.Context) {
	ctx := c.Request.Context()
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		softFail(c, 404, "内容不存在")
		return
	}
	identity := web.MustIdentity(c)

	var count int64
	if err := h.deps.DB.WithContext(ctx).Model(&store.Content{}).Where("id = ?", id).Count(&count).Error; err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	if count == 0 {
		softFail(c, 404, "内容不存在")
		return
	}

	reason := ""
	var body struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&body)
	reason = body.Reason

	claim := store.Claim{ContentID: id, UserID: identity.UID, Reason: &reason, Status: "pending"}
	if err := h.deps.DB.WithContext(ctx).Create(&claim).Error; err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	web.OK(c, gin.H{"id": claim.ID}, "认领申请已提交")
}

// toggleLike 切换点赞，并按推荐权重变化刷新推荐位与缓存。
func (h *Handler) toggleLike(c *gin.Context) {
	ctx := c.Request.Context()
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		web.Fail(c, 404, "内容不存在")
		return
	}
	identity := web.MustIdentity(c)

	var existing store.ContentLike
	err = h.deps.DB.WithContext(ctx).Where("content_id = ? AND user_id = ?", id, identity.UID).First(&existing).Error
	liked := true
	message := "已点赞"
	switch {
	case err == nil:
		if err := h.deps.DB.WithContext(ctx).Delete(&store.ContentLike{}, existing.ID).Error; err != nil {
			web.Fail(c, 500, "服务异常")
			return
		}
		liked = false
		message = "已取消点赞"
	case errors.Is(err, gorm.ErrRecordNotFound):
		like := store.ContentLike{ContentID: id, UserID: identity.UID}
		if err := h.deps.DB.WithContext(ctx).Create(&like).Error; err != nil {
			web.Fail(c, 500, "服务异常")
			return
		}
	default:
		web.Fail(c, 500, "服务异常")
		return
	}

	count := h.likeCount(ctx, id)
	// 点赞权重高，变化后立即刷新推荐位（Redis 锁防抖）。
	h.rec.RefreshAsync()
	h.invalidate(ctx, id)
	web.OK(c, gin.H{"liked": liked, "like_count": count}, message)
}

// likeStatus 返回当前用户的点赞/收藏状态与点赞总数。
func (h *Handler) likeStatus(c *gin.Context) {
	ctx := c.Request.Context()
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		web.Fail(c, 404, "内容不存在")
		return
	}
	identity := web.MustIdentity(c)

	var likes, favs int64
	h.deps.DB.WithContext(ctx).Model(&store.ContentLike{}).
		Where("content_id = ? AND user_id = ?", id, identity.UID).Count(&likes)
	h.deps.DB.WithContext(ctx).Model(&store.ContentFavorite{}).
		Where("content_id = ? AND user_id = ?", id, identity.UID).Count(&favs)

	web.OK(c, gin.H{
		"liked":      likes > 0,
		"favorited":  favs > 0,
		"like_count": h.likeCount(ctx, id),
	}, "ok")
}

// toggleFavorite 切换收藏。
func (h *Handler) toggleFavorite(c *gin.Context) {
	ctx := c.Request.Context()
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		web.Fail(c, 404, "内容不存在")
		return
	}
	identity := web.MustIdentity(c)

	var existing store.ContentFavorite
	err = h.deps.DB.WithContext(ctx).Where("content_id = ? AND user_id = ?", id, identity.UID).First(&existing).Error
	favorited := true
	message := "已收藏"
	switch {
	case err == nil:
		if err := h.deps.DB.WithContext(ctx).Delete(&store.ContentFavorite{}, existing.ID).Error; err != nil {
			web.Fail(c, 500, "服务异常")
			return
		}
		favorited = false
		message = "已取消收藏"
	case errors.Is(err, gorm.ErrRecordNotFound):
		fav := store.ContentFavorite{ContentID: id, UserID: identity.UID}
		if err := h.deps.DB.WithContext(ctx).Create(&fav).Error; err != nil {
			web.Fail(c, 500, "服务异常")
			return
		}
	default:
		web.Fail(c, 500, "服务异常")
		return
	}
	web.OK(c, gin.H{"favorited": favorited}, message)
}

// createContent 落库并返回装饰后的内容（含异步媒体处理与推荐刷新）。
func (h *Handler) createContent(ctx context.Context, in createInput) (Item, error) {
	row := store.Content{
		Title:       in.Title,
		FileSize:    in.FileSize,
		UserID:      in.UserID,
		Tags:        marshalTags(in.Tags),
		AuditStatus: in.AuditStatus,
	}
	if in.Content != "" {
		row.Content = &in.Content
	}
	if in.FilePath != "" {
		row.FilePath = &in.FilePath
	}
	if in.GuestNickname != "" {
		row.GuestNickname = &in.GuestNickname
	}
	if in.GuestEmail != "" {
		row.GuestEmail = &in.GuestEmail
	}

	db := h.deps.DB.WithContext(ctx)
	if err := db.Create(&row).Error; err != nil {
		slog.Error("内容入库失败", "err", err, "title", in.Title, "audit", in.AuditStatus)
		return Item{}, err
	}

	// 新内容改变所有列表/搜索/标签结果。
	h.deps.Redis.ClearContentListCache(ctx)

	if in.AbsPath != "" {
		go h.processMedia(row.ID, in.AbsPath, MediaTypeForPath(in.AbsPath))
	}
	if in.AuditStatus == "approved" {
		h.rec.RefreshAsync()
	}

	userMap := map[uint64]store.User{}
	if row.UserID > 0 {
		var u store.User
		if err := db.First(&u, row.UserID).Error; err == nil {
			userMap[u.ID] = u
		}
	}
	return decorate(row, userMap, 0, false), nil
}

// processMedia 异步生成缩略图并回写 thumb_path（失败仅告警，不影响上传结果）。
func (h *Handler) processMedia(id uint64, absPath, contentType string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	rel, err := media.GenerateThumbnail(ctx, absPath, contentType, h.deps.Cfg.ThumbDir)
	if err != nil {
		slog.Warn("缩略图生成失败", "id", id, "err", err)
		return
	}
	if err := h.deps.DB.WithContext(ctx).Model(&store.Content{}).
		Where("id = ?", id).Update("thumb_path", rel).Error; err != nil {
		slog.Warn("缩略图回写失败", "id", id, "err", err)
		return
	}
	h.invalidate(ctx, id)
}

// prepareUploadFile 返回上传文件的相对路径、大小与绝对路径。
// 保留原始格式不做转码（压缩交给后台 TinyPNG 任务，压缩后后缀不变）。
func (h *Handler) prepareUploadFile(f *uploadedFile) (string, int64, string) {
	if f == nil {
		return "", 0, ""
	}
	return f.RelPath, f.Size, f.AbsPath
}

// invalidate 失效单条详情与列表缓存（写路径统一调用）。
func (h *Handler) invalidate(ctx context.Context, id uint64) {
	h.deps.Redis.ClearContentCache(ctx, id)
	h.deps.Redis.ClearContentListCache(ctx)
}

func (h *Handler) likeCount(ctx context.Context, id uint64) int64 {
	var n int64
	h.deps.DB.WithContext(ctx).Model(&store.ContentLike{}).Where("content_id = ?", id).Count(&n)
	return n
}

func validTitle(title string) bool {
	n := len([]rune(title))
	return n >= 1 && n <= 200
}

func marshalTags(tags []string) string {
	if len(tags) == 0 {
		return "[]"
	}
	b, err := json.Marshal(tags)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func clientIP(c *gin.Context) string {
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		if i := strings.Index(xff, ","); i >= 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	return c.ClientIP()
}

func removeFile(path string) error {
	if path == "" {
		return nil
	}
	return os.Remove(path)
}
