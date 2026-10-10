package comment

import (
	"sort"
	"time"

	"github.com/huntersxy/xqecz/server/internal/store"
)

// buildThreads 将任意深度回复铺在所属顶层的一层 replies 中，保留直接 parent_id 和引用。
// 广度遍历避免深链的递归栈溢出；seen 同时防御损坏数据中的环或重复记录。
func buildThreads(tops, replies []store.Comment, users map[uint64]store.User) []TopDTO {
	children := map[uint64][]store.Comment{}
	for _, row := range replies {
		if row.ParentID != nil {
			children[*row.ParentID] = append(children[*row.ParentID], row)
		}
	}
	list := make([]TopDTO, 0, len(tops))
	seen := map[uint64]bool{}
	for _, top := range tops {
		seen[top.ID] = true
	}
	for _, top := range tops {
		thread := TopDTO{DTO: toDTO(top, users), Replies: []DTO{}}
		queue := []store.Comment{top}
		for head := 0; head < len(queue); head++ {
			parent := queue[head]
			for _, child := range children[parent.ID] {
				if seen[child.ID] || child.ContentID != top.ContentID {
					continue
				}
				seen[child.ID] = true
				dto := toDTO(child, users)
				parentDTO := toDTO(parent, users)
				dto.Parent = &ParentDTO{ID: parent.ID, UserID: parent.UserID, Text: parent.Text, User: parentDTO.User}
				thread.Replies = append(thread.Replies, dto)
				queue = append(queue, child)
			}
		}
		sort.Slice(thread.Replies, func(i, j int) bool {
			a, b := thread.Replies[i], thread.Replies[j]
			if time.Time(a.CreatedAt).Equal(time.Time(b.CreatedAt)) {
				return a.ID < b.ID
			}
			return time.Time(a.CreatedAt).Before(time.Time(b.CreatedAt))
		})
		list = append(list, thread)
	}
	return list
}
