package qiniu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// fusionHost 是 CDN（fusion）侧接口的接入域名。
// 与 kodo 的 api.qiniuapi.com 分属两套鉴权，见 qbox.go。
const fusionHost = "https://fusion.qiniuapi.com"

// cdnFlow 查询 CDN 加速域名在 [begin,end) 内的**计量流量**（账单口径，单位 Byte）。
//
// 用 /v2/tune/flux（计量）而非 /v2/tune/monitoring/flow（监控）：
// 免费额度是按计量数据抵扣的，拿监控数据去判 90% 会对不上账单。
func (c *Client) cdnFlow(ctx context.Context, begin, end time.Time) (int64, error) {
	if c.cdnDomain == "" {
		return 0, nil
	}
	body := map[string]any{
		"domains":     c.cdnDomain, // 用量统计接口的 domains 是**分号分隔的字符串**，不是数组
		"startDate":   begin.In(cst).Format("2006-01-02"),
		"endDate":     end.In(cst).Format("2006-01-02"),
		"granularity": "day",
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fusionHost+"/v2/tune/flux", bytes.NewReader(raw))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.signer.AuthorizeQbox(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer drain(resp)
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("qiniu fusion /v2/tune/flux: status=%d", resp.StatusCode)
	}

	// 返回形如 {"data":{"<date>":{"flux":N}}|["..."]}——七牛的用量返回随接口版本
	// 有过调整，这里只抽取**数值叶子的总和**，不锁死一层结构。
	var payload any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, err
	}
	return sumNumbers(payload), nil
}

// sumNumbers 深度遍历 JSON，把所有整数值累加。
// 用量返回里除流量数字外只剩日期串与域名串，累加数值即为总量；
// 这样对字段名与嵌套层数的变化不敏感，避免七牛改个壳就判 0。
func sumNumbers(v any) int64 {
	var sum int64
	switch t := v.(type) {
	case float64:
		sum += int64(t)
	case []any:
		for _, e := range t {
			sum += sumNumbers(e)
		}
	case map[string]any:
		for _, e := range t {
			sum += sumNumbers(e)
		}
	}
	return sum
}
