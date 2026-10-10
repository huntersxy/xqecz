// Package avatar 根据邮箱生成公开头像地址，不在公开响应中暴露邮箱。
package avatar

import (
	"crypto/md5"
	"encoding/hex"
	"regexp"
	"strconv"
	"strings"
)

var qqMailPattern = regexp.MustCompile(`^(\d{5,11})@qq\.com$`)

// URL 为 QQ 邮箱返回 QQ 头像，其余邮箱返回 Gravatar，未设置邮箱时返回空串。
func URL(email string, size int) string {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" {
		return ""
	}
	if m := qqMailPattern.FindStringSubmatch(normalized); len(m) == 2 {
		return "https://q.qlogo.cn/headimg_dl?dst_uin=" + m[1] + "&spec=100"
	}
	sum := md5.Sum([]byte(normalized))
	return "https://www.gravatar.com/avatar/" + hex.EncodeToString(sum[:]) + "?d=identicon&s=" + strconv.Itoa(size)
}
