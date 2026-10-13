package aigen

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	core "github.com/huntersxy/xqecz/server/internal/aigen"
	"github.com/huntersxy/xqecz/server/internal/store"
)

// The plaza batches authors, assets and prices; its cost never requires one
// network/database round trip per card. Sponsorship revalidates authoritatively.
func (h *Handler) ideaViews(c *gin.Context, rows []core.Idea) ([]any, error) {
	userIDs, assetIDs, taskIDs := []uint64{}, []uint64{}, []uint64{}
	inputs := map[uint64]core.Input{}
	for _, row := range rows {
		userIDs = append(userIDs, row.UserID, row.SponsorID)
		var input core.Input
		_ = json.Unmarshal([]byte(row.Snapshot), &input)
		inputs[row.ID] = input
		for _, ref := range input.References {
			assetIDs = append(assetIDs, ref.AssetID)
		}
		if row.Status == "realized" {
			taskIDs = append(taskIDs, row.TaskID)
		}
	}
	users := []store.User{}
	if len(userIDs) > 0 {
		if err := h.db(c).Select("id,username").Where("id IN ?", userIDs).Find(&users).Error; err != nil {
			return nil, err
		}
	}
	names := map[uint64]string{}
	for _, user := range users {
		names[user.ID] = user.Username
	}
	author := func(id uint64) any {
		name := names[id]
		if name == "" {
			name = "已注销用户"
		}
		return gin.H{"id": id, "username": name}
	}
	assets := []core.Asset{}
	if len(assetIDs) > 0 || len(taskIDs) > 0 {
		if err := h.db(c).Where("id IN ? OR task_id IN ?", assetIDs, taskIDs).Find(&assets).Error; err != nil {
			return nil, err
		}
	}
	byID := map[uint64]core.Asset{}
	byTask := map[uint64][]core.Asset{}
	for _, asset := range assets {
		byID[asset.ID] = asset
		byTask[asset.TaskID] = append(byTask[asset.TaskID], asset)
	}
	providers := []core.ProviderConfig{}
	if err := h.db(c).Where("user_id = 0 AND enabled = ? AND cipher_key <> ?", true, "").Order("`default` desc,id asc").Find(&providers).Error; err != nil {
		return nil, err
	}
	byProvider := map[uint64]core.ProviderConfig{}
	for _, provider := range providers {
		byProvider[provider.ID] = provider
	}
	out := []any{}
	for _, row := range rows {
		input := inputs[row.ID]
		refs, results := []core.Asset{}, []core.Asset{}
		for _, ref := range input.References {
			if asset, ok := byID[ref.AssetID]; ok {
				refs = append(refs, asset)
			}
		}
		if row.ReferenceSnapshot != "" {
			var snapshot []core.Asset
			if json.Unmarshal([]byte(row.ReferenceSnapshot), &snapshot) == nil {
				refs = snapshot
			}
		}
		if row.Status == "realized" {
			results = byTask[row.TaskID]
			if results == nil {
				results = []core.Asset{}
			}
		}
		provider, available := byProvider[input.ProviderID]
		if input.ProviderID == 0 && len(providers) > 0 {
			provider, available = providers[0], true
		}
		cost := row.EstimatedCost
		if available {
			cost = provider.Cost
			var prices map[string]int64
			_ = json.Unmarshal([]byte(provider.QualityCosts), &prices)
			if price := prices[input.Quality]; price > 0 {
				cost = price
			}
			cost *= int64(input.Count)
		}
		allowed := available && provider.Model == row.Model && input.Count <= provider.MaxCount && len(input.References) <= provider.MaxReferences
		view := gin.H{"idea": row, "author": author(row.UserID), "sponsor": nil, "references": refs, "results": results, "current_cost": cost, "sponsorable": row.Status == "waiting" && allowed}
		if row.SponsorID != 0 {
			view["sponsor"] = author(row.SponsorID)
		}
		out = append(out, view)
	}
	return out, nil
}
