package aigen

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const DailyCreditCap int64 = 5

var rewardLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

// GrantDailyLogin also runs when a cookie session is restored. The account lock
// serializes tabs and login retries; reserved daily credits still occupy the cap
// so a failed generation can return its original credits without losing any.
func GrantDailyLogin(ctx context.Context, db *gorm.DB, uid uint64, now time.Time) error {
	day := now.In(rewardLocation).Format("2006-01-02")
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		account, err := lockAccount(tx, uid)
		if err != nil {
			return err
		}
		if account.LastLoginDay >= day {
			return nil
		}
		amount := int64(0)
		if account.DailyAvailable+account.DailyReserved < DailyCreditCap {
			amount = 1
		}
		if err := tx.Model(&Account{}).Where("user_id = ?", uid).Updates(map[string]any{
			"last_login_day": day, "available": account.Available + amount,
			"daily_available": account.DailyAvailable + amount,
		}).Error; err != nil {
			return err
		}
		if amount == 0 {
			return nil
		}
		return tx.Create(&Ledger{UserID: uid, Key: fmt.Sprintf("daily-login:%d:%s", uid, day), Kind: "daily_login", Amount: amount, Note: "每日登录"}).Error
	})
}

// RewardApproval belongs to the same transaction as the audit status update.
// A content ID earns once even after rejection, reapproval, or author changes.
func RewardApproval(tx *gorm.DB, uid, contentID uint64) error {
	if uid == 0 {
		return nil
	}
	account, err := lockAccount(tx, uid)
	if err != nil {
		return err
	}
	receipt := Ledger{UserID: uid, Key: fmt.Sprintf("content-approved:%d", contentID), Kind: "content_approved", Amount: 1, Note: fmt.Sprintf("作品 #%d 审核通过", contentID)}
	result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&receipt)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return nil
	}
	return tx.Model(&Account{}).Where("user_id = ?", uid).Update("available", account.Available+1).Error
}
