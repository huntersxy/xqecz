package server

import (
	"math"
	"slices"
	"time"

	pb "xqecz-worker/proto"
)

// timeDecayScore 时间衰减分数：1 天内 100，7 天内线性衰减到 0，更久为 0。
func timeDecayScore(createdAtUnix int64) float64 {
	t := time.Unix(createdAtUnix, 0)
	daysAgo := time.Since(t).Hours() / 24
	switch {
	case daysAgo < 1:
		return 100.0
	case daysAgo < 7:
		return 50.0 * (1.0 - daysAgo/7.0)
	default:
		return 0.0
	}
}

// scoreItem 综合「时间衰减」与「浏览量」为单条内容打分（纯函数，无外部依赖）。
func scoreItem(item *pb.RecommendItem) float64 {
	timeScore := timeDecayScore(item.GetCreatedAtUnix())
	// 浏览量信号：封顶 1000，权重 50。
	viewScore := math.Min(float64(item.GetViewCount()), 1000) / 1000.0 * 50.0
	return timeScore + viewScore
}

// computeRecommend 对输入列表逐个打分，返回按分数降序排列的评分结果。
func computeRecommend(items []*pb.RecommendItem) []*pb.ScoredItem {
	out := make([]*pb.ScoredItem, 0, len(items))
	for _, it := range items {
		if it == nil {
			continue
		}
		out = append(out, &pb.ScoredItem{
			ContentId: it.GetContentId(),
			Score:     scoreItem(it),
		})
	}
	slices.SortFunc(out, func(a, b *pb.ScoredItem) int {
		sa, sb := a.GetScore(), b.GetScore()
		if sa > sb {
			return -1
		}
		if sa < sb {
			return 1
		}
		return 0
	})
	return out
}
