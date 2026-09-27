package qiniu

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// cst 是七牛统计与计费所依据的时区（北京时间）。
// 自然月的边界按它划分，与部署机的本地时区无关——
// 机器漂到 UTC 也不会把「当月」算错一个月。
var cst = time.FixedZone("CST", 8*3600)

const statsHost = "https://api.qiniuapi.com"

// Config 是统计客户端的配置。
type Config struct {
	Signer Signer
	// Bucket 是空间名，用于把统计口径限定在本空间（账号下有多个空间时必须给）。
	Bucket string
	// CDNDomain 是 CDN 加速域名；留空表示不查「CDN 下载流量」。
	// 注意这不是「该指标用量为 0」，而是「该指标不参与判定」：
	// 没绑域名就没有 CDN 流量，二者是一致的。
	CDNDomain string
	Timeout   time.Duration
}

// Client 查询七牛用量统计；零值不可用，须经 New 构造。
type Client struct {
	signer    Signer
	bucket    string
	cdnDomain string
	http      *http.Client
}

// Usage 是一个自然月内（月首至今）的用量。
type Usage struct {
	// StorageBytes 是**时点值**：当前存储量，不是当月累计。
	StorageBytes int64
	// OriginFlowBytes 是 CDN 回源流出，当月累计。
	OriginFlowBytes int64
	// EgressBytes 是外网流出（用户直连源站），当月累计。
	// 这一项**没有免费额度**，因此不看百分比、只看绝对量。
	EgressBytes int64
	// CDNFlowBytes 是 CDN 下载流量，当月累计。
	CDNFlowBytes int64
	// GETCount / PUTCount 是当月请求次数，当月累计。
	GETCount int64
	PUTCount int64
	// CDNKnown 表示 CDNFlowBytes 本次是否真的取到。
	// 取不到（未绑域名、或查询失败）时该字段为 false，判定时须跳过，
	// 绝不能把「查不到」当成「用量为 0」。
	CDNKnown bool
}

// New 构造统计客户端。
func New(cfg Config) (*Client, error) {
	if !cfg.Signer.Enabled() {
		return nil, fmt.Errorf("qiniu: 凭据不完整")
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	return &Client{
		signer:    cfg.Signer,
		bucket:    cfg.Bucket,
		cdnDomain: strings.Trim(strings.TrimSpace(cfg.CDNDomain), "/"),
		http:      &http.Client{Timeout: timeout},
	}, nil
}

// MonthWindow 返回 now 所在自然月的 [月首, now]，按北京时间划分。
// end 用「一小时后」留出余量，避免恰好卡在月初边界时漏掉最后一段。
func MonthWindow(now time.Time) (begin, end time.Time) {
	local := now.In(cst)
	begin = time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, cst)
	return begin, local.Add(time.Hour)
}

// Usage 查询当月用量。任一必要项失败即整体返回错误——
// 调用方据此保留上一次的结论，绝不拿半份数据去改闸门。
func (c *Client) Usage(ctx context.Context, now time.Time) (Usage, error) {
	var u Usage
	begin, end := MonthWindow(now)

	base := func(extra url.Values) url.Values {
		vals := url.Values{}
		vals.Set("begin", begin.Format("20060102150405"))
		vals.Set("end", end.Format("20060102150405"))
		vals.Set("g", "day")
		if c.bucket != "" {
			vals.Set("$bucket", c.bucket)
		}
		for k, v := range extra {
			vals[k] = v
		}
		return vals
	}

	var err error
	// 累计量（流量/请求次数）按天求和 = 月首至今。
	if u.OriginFlowBytes, err = c.flow(ctx, base(url.Values{
		"$metric": []string{"cdn_flow_out"}, "select": []string{"flow"},
	})); err != nil {
		return Usage{}, fmt.Errorf("CDN 回源流量: %w", err)
	}
	if u.EgressBytes, err = c.flow(ctx, base(url.Values{
		"$metric": []string{"flow_out"}, "select": []string{"flow"},
	})); err != nil {
		return Usage{}, fmt.Errorf("外网流出流量: %w", err)
	}
	if u.GETCount, err = c.hits(ctx, base(url.Values{
		"$metric": []string{"hits"}, "select": []string{"hits"},
	})); err != nil {
		return Usage{}, fmt.Errorf("GET 次数: %w", err)
	}
	// 存储量是时点值，取最后一个数据点而非求和。
	if u.StorageBytes, err = c.storage(ctx, base(url.Values{"select": []string{"size"}})); err != nil {
		return Usage{}, fmt.Errorf("存储量: %w", err)
	}
	// PUT 次数：rs_put 同样按天返回，求和即月累计。
	if u.PUTCount, err = c.rsPut(ctx, base(url.Values{"select": []string{"count"}})); err != nil {
		return Usage{}, fmt.Errorf("PUT 次数: %w", err)
	}

	if c.cdnDomain != "" {
		flow, err := c.cdnFlow(ctx, begin, end)
		if err != nil {
			return Usage{}, fmt.Errorf("CDN 下载流量: %w", err)
		}
		u.CDNFlowBytes, u.CDNKnown = flow, true
	}
	return u, nil
}

/* ---------------- kodo 统计接口 ---------------- */

// flow 解析 blob_io 的 [{time,values:{flow}}]，按天求和。
func (c *Client) flow(ctx context.Context, vals url.Values) (int64, error) {
	var rows []struct {
		Values struct {
			Flow int64 `json:"flow"`
		} `json:"values"`
	}
	if err := c.getJSON(ctx, "/v6/blob_io", vals, &rows); err != nil {
		return 0, err
	}
	var sum int64
	for _, r := range rows {
		sum += r.Values.Flow
	}
	return sum, nil
}

// hits 解析 blob_io 的 GET 次数（select=hits → values.hits）。
func (c *Client) hits(ctx context.Context, vals url.Values) (int64, error) {
	var rows []struct {
		Values struct {
			Hits int64 `json:"hits"`
		} `json:"values"`
	}
	if err := c.getJSON(ctx, "/v6/blob_io", vals, &rows); err != nil {
		return 0, err
	}
	var sum int64
	for _, r := range rows {
		sum += r.Values.Hits
	}
	return sum, nil
}

// rsPut 解析 /v6/rs_put 的 PUT 次数：响应形状与 blob_io 相同（[{time,values}]）。
func (c *Client) rsPut(ctx context.Context, vals url.Values) (int64, error) {
	var rows []struct {
		Values struct {
			Count int64 `json:"count"`
		} `json:"values"`
	}
	if err := c.getJSON(ctx, "/v6/rs_put", vals, &rows); err != nil {
		return 0, err
	}
	var sum int64
	for _, r := range rows {
		sum += r.Values.Count
	}
	return sum, nil
}

// storage 解析 /v6/space 的 {"times":[],"datas":[]}，取最后一个数据点。
// 存储量是水平值（当前有多少字节），求和会把每天的快照叠加成天文数字。
func (c *Client) storage(ctx context.Context, vals url.Values) (int64, error) {
	var payload struct {
		Datas []int64 `json:"datas"`
	}
	if err := c.getJSON(ctx, "/v6/space", vals, &payload); err != nil {
		return 0, err
	}
	if len(payload.Datas) == 0 {
		return 0, nil
	}
	return payload.Datas[len(payload.Datas)-1], nil
}

// getJSON 发一个带管理凭证的 GET 并把响应体解到 out。
func (c *Client) getJSON(ctx context.Context, path string, vals url.Values, out any) error {
	u := statsHost + path
	if enc := vals.Encode(); enc != "" {
		u += "?" + enc
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	c.signer.Authorize(req, nil)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer drain(resp)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("qiniu GET %s: status=%d %s", path, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func drain(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 8192))
		_ = resp.Body.Close()
	}
}
