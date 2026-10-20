package aigen

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

type receiptAsset struct {
	Asset   Asset  `json:"asset"`
	File    string `json:"file"`
	Preview string `json:"preview"`
	Digest  string `json:"digest"`
}
type resultReceipt struct {
	TaskID  uint64         `json:"task_id"`
	UserID  uint64         `json:"user_id"`
	Status  string         `json:"status"`
	Message string         `json:"message"`
	Assets  []receiptAsset `json:"assets"`
}

func (s *Service) receiptPath(id uint64) string {
	return filepath.Join(s.dir, fmt.Sprintf(".task-%d-result.json", id))
}

func (s *Service) writeReceipt(task Task, status, message string, assets []Asset) error {
	receipt := resultReceipt{TaskID: task.ID, UserID: task.UserID, Status: status, Message: message, Assets: []receiptAsset{}}
	for _, asset := range assets {
		receipt.Assets = append(receipt.Assets, receiptAsset{Asset: asset, File: asset.FileName, Preview: asset.PreviewFileName, Digest: asset.Digest})
	}
	f, err := os.CreateTemp(s.dir, ".result-*")
	if err != nil {
		return err
	}
	temp := f.Name()
	defer os.Remove(temp)
	if err = json.NewEncoder(f).Encode(receipt); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(temp, s.receiptPath(task.ID))
}
func (s *Service) readReceipt(task Task) (string, string, []Asset, error) {
	f, err := os.Open(s.receiptPath(task.ID))
	if err != nil {
		return "", "", nil, err
	}
	defer f.Close()
	var receipt resultReceipt
	if err = json.NewDecoder(io.LimitReader(f, 64<<10)).Decode(&receipt); err != nil {
		return "", "", nil, err
	}
	if receipt.TaskID != task.ID || receipt.UserID != task.UserID || (receipt.Status != "succeeded" && receipt.Status != "unknown") || len(receipt.Assets) > 2 || (receipt.Status == "succeeded" && len(receipt.Assets) != task.Count) {
		return "", "", nil, ErrInvalid
	}
	assets := []Asset{}
	for _, saved := range receipt.Assets {
		asset := saved.Asset
		asset.FileName, asset.PreviewFileName, asset.Digest = saved.File, saved.Preview, saved.Digest
		if asset.TaskID != task.ID || asset.UserID != task.UserID {
			return "", "", nil, ErrInvalid
		}
		path, err := s.AssetPath(asset)
		if err != nil {
			return "", "", nil, err
		}
		stat, err := os.Stat(path)
		if err != nil || !stat.Mode().IsRegular() || stat.Size() != asset.Bytes {
			return "", "", nil, ErrInvalid
		}
		assets = append(assets, asset)
	}
	return receipt.Status, receipt.Message, assets, nil
}
func (s *Service) removeReceipt(id uint64) {
	if err := os.Remove(s.receiptPath(id)); err != nil && !errors.Is(err, os.ErrNotExist) {
		slog.Warn("AI committed receipt cleanup failed", "task_id", id)
	}
}
