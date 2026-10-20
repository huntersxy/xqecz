package avatar

import (
	"crypto/md5"
	"encoding/hex"
	"regexp"
	"strconv"
	"strings"
)

var qqMailPattern = regexp.MustCompile(`^(\d{5,11})@qq\.com$`)

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
