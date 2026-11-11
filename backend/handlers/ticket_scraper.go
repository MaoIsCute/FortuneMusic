package handlers

import (
	"net/http"
	"time"

	"fortune-tracker/db"
	"fortune-tracker/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ticketSlotInput struct {
	EventDate  string `json:"event_date"`
	Session    string `json:"session"`
	TimeRange  string `json:"time_range"`
	MemberName string `json:"member_name"`
	HeldCount  int    `json:"held_count"`
	UsedCount  int    `json:"used_count"`
}

type ticketInput struct {
	Group        string            `json:"group"`
	SingleNumber int               `json:"single_number"`
	SingleName   string            `json:"single_name"`
	Slots        []ticketSlotInput `json:"slots"`
}

// PushTicketHoldings 接收擴充功能抓到的 forTUNE meets 券面頁（保有数快照）。
// 每張單曲整份覆蓋：先刪掉該使用者該 (group, single_number) 的舊列再寫入新列，
// 同一個 transaction 裡做，不會出現刪了一半的狀態。券面上某一格消失（例如之後被
// 轉讓/取消）也會跟著消失，這是快照語意要的行為。
func PushTicketHoldings(c *gin.Context) {
	var req struct {
		ScrapeToken string        `json:"scrape_token" binding:"required"`
		Tickets     []ticketInput `json:"tickets"`
		// nil = 擴充功能沒讀到未付款清單（讀取失敗或舊版），維持資料庫原狀不覆蓋；
		// 空陣列 = 讀到了、確實沒有未付款的單，要清空
		UnpaidApplySeqs *[]string `json:"unpaid_apply_seqs"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "請提供 scrape_token 與 tickets"})
		return
	}

	var user models.User
	if err := db.DB.Where("scrape_token = ?", req.ScrapeToken).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "無效的 scrape token"})
		return
	}

	now := time.Now()
	singles, slots := 0, 0
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		for _, t := range req.Tickets {
			// 解析不到團體或單曲號的券面（專輯、或網站改版導致解析失敗）直接跳過，
			// 不能用空 group/0 當鍵寫入，否則不同券面會互相覆蓋
			if t.Group == "" || t.SingleNumber <= 0 {
				continue
			}
			if err := tx.Where(`user_id = ? AND "group" = ? AND single_number = ?`, user.ID, t.Group, t.SingleNumber).
				Delete(&models.TicketHolding{}).Error; err != nil {
				return err
			}
			rows := make([]models.TicketHolding, 0, len(t.Slots))
			for _, s := range t.Slots {
				rows = append(rows, models.TicketHolding{
					UserID:       user.ID,
					Group:        t.Group,
					SingleNumber: t.SingleNumber,
					SingleName:   t.SingleName,
					EventDate:    normalizeEventDate(s.EventDate),
					Session:      s.Session,
					TimeRange:    s.TimeRange,
					MemberName:   normalizeMember(s.MemberName),
					HeldCount:    s.HeldCount,
					UsedCount:    s.UsedCount,
					ScrapedAt:    now,
				})
			}
			if len(rows) > 0 {
				if err := tx.Create(&rows).Error; err != nil {
					return err
				}
			}
			singles++
			slots += len(rows)
		}

		if req.UnpaidApplySeqs != nil {
			if err := tx.Where("user_id = ?", user.ID).Delete(&models.UnpaidApply{}).Error; err != nil {
				return err
			}
			unpaid := make([]models.UnpaidApply, 0, len(*req.UnpaidApplySeqs))
			for _, seq := range *req.UnpaidApplySeqs {
				if seq != "" {
					unpaid = append(unpaid, models.UnpaidApply{UserID: user.ID, ApplySeq: seq, ScrapedAt: now})
				}
			}
			if len(unpaid) > 0 {
				if err := tx.Create(&unpaid).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "寫入券面資料失敗"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"singles": singles, "slots": slots})
}
