package recommend

import (
	"testing"
	"time"
)

// TestScoreItem 覆盖打分的三段构成与饱和上限（点赞权重高于浏览量）。
func TestScoreItem(t *testing.T) {
	now := time.Now().Unix()

	if got := ScoreItem(Item{ContentID: 1, CreatedAtUnix: now}); got != 100 {
		t.Errorf("1 天内的新内容时间分应为 100，实际 %v", got)
	}

	old := time.Now().Add(-8 * 24 * time.Hour).Unix()
	if got := ScoreItem(Item{ContentID: 2, CreatedAtUnix: old}); got != 0 {
		t.Errorf("超过 7 天的内容时间分应为 0，实际 %v", got)
	}

	if got := ScoreItem(Item{ContentID: 3, CreatedAtUnix: now, ViewCount: 1000, LikeCount: 500}); got != 230 {
		t.Errorf("满分应为 230（100+30+100），实际 %v", got)
	}

	if got := ScoreItem(Item{ContentID: 4, CreatedAtUnix: now, ViewCount: 99999, LikeCount: 99999}); got != 230 {
		t.Errorf("超限输入应饱和在 230，实际 %v", got)
	}

	likes := ScoreItem(Item{ContentID: 5, CreatedAtUnix: now, LikeCount: 500})
	views := ScoreItem(Item{ContentID: 6, CreatedAtUnix: now, ViewCount: 1000})
	if likes <= views {
		t.Errorf("点赞权重应高于浏览量：likes=%v views=%v", likes, views)
	}
}

// TestComputeSortsDescending 校验输出按分数降序，与输入顺序无关。
func TestComputeSortsDescending(t *testing.T) {
	now := time.Now().Unix()
	out := Compute([]Item{
		{ContentID: 1, CreatedAtUnix: now, LikeCount: 10},
		{ContentID: 2, CreatedAtUnix: now, LikeCount: 400},
		{ContentID: 3, CreatedAtUnix: now, LikeCount: 0},
	})
	if len(out) != 3 {
		t.Fatalf("输出条数应为 3，实际 %d", len(out))
	}
	if out[0].ContentID != 2 {
		t.Errorf("点赞最多的内容应排首位，实际首位 id=%d", out[0].ContentID)
	}
	for i := 1; i < len(out); i++ {
		if out[i-1].Score < out[i].Score {
			t.Errorf("结果未按分数降序：%v < %v", out[i-1].Score, out[i].Score)
		}
	}
	if got := Compute(nil); len(got) != 0 {
		t.Errorf("空输入应返回空结果，实际 %v", got)
	}
}
