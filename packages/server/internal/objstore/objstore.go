// Package objstore 定义镜像目标共用的最小对象存储契约。
//
// 两个实现：internal/r2（S3 兼容，手写 SigV4）与 internal/openlist（HTTP API）。
// 契约刻意只有两个方法——镜像链只需要「写一份」和「问远端有没有同一份」，
// 多了就是给实现方发空头支票（OpenList 的 Local 驱动既没有桶也没有 ACL 概念）。
package objstore

import (
	"errors"
	"time"
)

// ErrCorrupted 表示本地文件在读取过程中被改写（最典型的是 TinyPNG 压缩任务的
// 就地替换）。调用方应丢弃本轮结果，下一轮自然会重传，不是故障。
var ErrCorrupted = errors.New("local file changed while reading")

// ObjectMeta 是远端对象的元数据。
//
// 只有 ContentLen 是所有实现都能给的字段：R2 从 HEAD 的 Content-Length 拿，
// OpenList 从 /api/fs/get 的 size 拿。MetaMD5 / ETag 是 S3 家族专有，
// OpenList 的 Local 驱动 hashinfo 恒为 "null"，故两者留空——
// 幂等判定必须容忍「远端没有哈希」这种情况（见 internal/mirror 的 sameObject）。
type ObjectMeta struct {
	ETag         string
	ContentLen   int64
	MetaMD5      string
	LastModified time.Time
}

// Store 是一个可写的对象存储目标。
//
// key 是桶内/挂载内的相对路径（不含前导斜杠），absPath 是待上传的本地绝对路径。
// Head 在对象不存在时返回 (零值, false, nil)——「不存在」不是错误。
type Store interface {
	Put(key, absPath, contentType string) error
	Head(key string) (ObjectMeta, bool, error)
}
