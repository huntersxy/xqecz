package content

import "testing"

// TestListCacheKey 校验列表缓存键的规范化：顺序无关、非法排序回落、参数不同则键不同。
func TestListCacheKey(t *testing.T) {
	a := listOpts{Page: 2, PageSize: 20, Tag: "x", AuditStatuses: []string{"pending", "approved"}, Keyword: "k", SortBy: "created_at", Order: "desc"}
	b := listOpts{Page: 2, PageSize: 20, Tag: "x", AuditStatuses: []string{"approved", "pending"}, Keyword: "k", SortBy: "created_at", Order: "desc"}
	if a.cacheKey() != b.cacheKey() {
		t.Error("审核状态集合顺序不同时应产生同一缓存键")
	}

	bad := listOpts{Page: 1, PageSize: 20, SortBy: "bogus", Order: ""}
	bad.normalize()
	good := listOpts{Page: 1, PageSize: 20, SortBy: "created_at", Order: "desc"}
	good.normalize()
	if bad.cacheKey() != good.cacheKey() {
		t.Error("非法排序参数应回落到默认值并产生同一缓存键")
	}

	other := listOpts{Page: 1, PageSize: 20, Keyword: "different"}
	other.normalize()
	if other.cacheKey() == good.cacheKey() {
		t.Error("不同查询参数不应共用缓存键")
	}
}

// TestNormalizeBounds 校验分页与排序参数的边界收敛。
func TestNormalizeBounds(t *testing.T) {
	o := listOpts{Page: 0, PageSize: 0, SortBy: "view_count", Order: "ASC"}
	o.normalize()
	if o.Page != 1 || o.PageSize != 20 {
		t.Errorf("非法分页应回落为 page=1 page_size=20，实际 page=%d size=%d", o.Page, o.PageSize)
	}
	if o.Order != "desc" {
		t.Errorf("非 asc 的排序方向应回落为 desc，实际 %q", o.Order)
	}

	big := listOpts{Page: 1, PageSize: 5000}
	big.normalize()
	if big.PageSize != 100 {
		t.Errorf("页大小应被限制在 100，实际 %d", big.PageSize)
	}
}
