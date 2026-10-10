package comment

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/huntersxy/xqecz/server/internal/store"
)

func commentRow(id, parent uint64, text string) store.Comment {
	row := store.Comment{ID: id, ContentID: 9, UserID: id, Text: text, CreatedAt: time.Unix(int64(id), 0)}
	if parent != 0 {
		row.ParentID = &parent
	}
	return row
}

func TestNestedRepliesRemainVisibleWithDirectParent(t *testing.T) {
	// 故障复现：A <- B <- C <- D，输入乱序且另有 A 的直接回复 E。
	a, b, c, d, e := commentRow(1, 0, "A"), commentRow(2, 1, "B"), commentRow(3, 2, "C"), commentRow(4, 3, "D"), commentRow(5, 1, "E")
	users := map[uint64]store.User{2: {ID: 2, Username: "作者B"}}
	threads := buildThreads([]store.Comment{a}, []store.Comment{d, e, c, b}, users)
	if len(threads) != 1 || len(threads[0].Replies) != 4 {
		t.Fatalf("完整楼内回复应返回 B/C/D/E: %+v", threads)
	}
	replies := threads[0].Replies
	for i, want := range []uint64{2, 3, 4, 5} {
		if replies[i].ID != want {
			t.Fatalf("回复顺序错误: %+v", replies)
		}
	}
	if *replies[1].ParentID != b.ID || replies[1].Parent.ID != b.ID || replies[1].Parent.Text != "B" || replies[1].Parent.User.Username != "作者B" {
		t.Fatalf("C 必须保留 B 的引用，而非被改为回复 A: %+v", replies[1])
	}
}

func TestThreadsRespectPageAndContentBoundary(t *testing.T) {
	a := commentRow(1, 0, "A")
	foreign := commentRow(4, 1, "跨内容")
	foreign.ContentID = 10
	rows := []store.Comment{commentRow(2, 1, "B"), commentRow(3, 2, "C"), foreign, commentRow(6, 5, "另一页的回复"), commentRow(8, 99, "断链回复")}
	threads := buildThreads([]store.Comment{a}, rows, nil)
	if len(threads[0].Replies) != 2 {
		t.Fatalf("只返回本页本内容的完整回复链: %+v", threads)
	}
	// 删除 B 后 C 被提升到顶层，D 仍显示在 C 下。
	c := commentRow(3, 0, "C")
	threads = buildThreads([]store.Comment{c}, []store.Comment{commentRow(7, 3, "D")}, nil)
	if len(threads) != 1 || len(threads[0].Replies) != 1 || threads[0].Replies[0].Parent.ID != 3 {
		t.Fatal("删除父评论后回复链必须仍可展示")
	}
}

func TestDeepReplyChainAndDuplicateRows(t *testing.T) {
	const depth = 2000
	rows := make([]store.Comment, 0, depth)
	for id := uint64(2); id <= depth; id++ {
		rows = append(rows, commentRow(id, id-1, "reply"))
	}
	rows = append(rows, rows[0], commentRow(1, depth, "cycle"))
	threads := buildThreads([]store.Comment{commentRow(1, 0, "A")}, rows, nil)
	if len(threads[0].Replies) != depth-1 {
		t.Fatalf("深链或重复记录导致丢失/重复: %d", len(threads[0].Replies))
	}
}

func TestEmptyRepliesMarshalAsArray(t *testing.T) {
	threads := buildThreads([]store.Comment{commentRow(1, 0, "A")}, nil, nil)
	data, err := json.Marshal(threads)
	if err != nil || !strings.Contains(string(data), "\"replies\":[]") {
		t.Fatalf("前端契约要求空数组: %s, %v", data, err)
	}
}

func TestCommentAvatarsDoNotExposeEmail(t *testing.T) {
	qq, gravatar := "12345@qq.com", "Test@Example.com"
	users := map[uint64]store.User{
		1: {ID: 1, Username: "QQ作者", Email: &qq},
		2: {ID: 2, Username: "Gravatar作者", Email: &gravatar},
		3: {ID: 3, Username: "未设置邮箱"},
	}
	threads := buildThreads([]store.Comment{commentRow(1, 0, "A")}, []store.Comment{commentRow(2, 1, "B"), commentRow(3, 2, "C"), commentRow(4, 3, "D")}, users)
	thread := threads[0]
	if thread.User.AvatarURL != "https://q.qlogo.cn/headimg_dl?dst_uin=12345&spec=100" {
		t.Fatalf("顶层评论缺少 QQ 头像: %+v", thread.User)
	}
	if thread.Replies[0].User.AvatarURL != "https://www.gravatar.com/avatar/55502f40dc8b7c769880b10874abc9d0?d=identicon&s=80" {
		t.Fatalf("回复缺少 Gravatar 头像: %+v", thread.Replies[0].User)
	}
	if thread.Replies[0].Parent.User.AvatarURL != thread.User.AvatarURL {
		t.Fatal("直接回复引用丢失作者头像")
	}
	for _, r := range thread.Replies[1:] {
		if r.User.AvatarURL != "" {
			t.Fatalf("缺失邮箱或作者不应生成远端头像: %+v", r.User)
		}
	}
	raw, err := json.Marshal(threads)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "@") || strings.Contains(string(raw), "\"email\"") {
		t.Fatalf("公开评论响应泄漏邮箱: %s", raw)
	}
}
