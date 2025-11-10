package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"kbtg-ai-workshop-nov/workshop-4/backend/internal/models"
	"kbtg-ai-workshop-nov/workshop-4/backend/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.Engine) {
	r.GET("/users", getUsers)
	r.GET("/users/:id", getUser)
	r.POST("/users", createUser)
	r.PUT("/users/:id", updateUser)
	r.DELETE("/users/:id", deleteUser)

	r.POST("/transfers", createTransfer)
	r.GET("/transfers", listTransfers)
	r.GET("/transfers/:id", getTransfer)
}

func getUsers(c *gin.Context) {
	var users []models.User
	db := store.GetDB()
	if err := db.Order("id desc").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, users)
}

func getUser(c *gin.Context) {
	id := c.Param("id")
	var u models.User
	db := store.GetDB()
	if err := db.First(&u, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, u)
}

func createUser(c *gin.Context) {
	var u models.User
	if err := c.BindJSON(&u); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if u.RegistrationDate == "" {
		u.RegistrationDate = time.Now().Format("2006-01-02")
	}
	db := store.GetDB()
	if err := db.Create(&u).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, u)
}

func updateUser(c *gin.Context) {
	id := c.Param("id")
	var u models.User
	if err := c.BindJSON(&u); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	db := store.GetDB()
	var existing models.User
	if err := db.First(&existing, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	u.ID = existing.ID
	if err := db.Save(&u).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, u)
}

func deleteUser(c *gin.Context) {
	id := c.Param("id")
	db := store.GetDB()
	if err := db.Delete(&models.User{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// Transfer handlers
type transferCreateReq struct {
	FromUserID uint   `json:"fromUserId" binding:"required"`
	ToUserID   uint   `json:"toUserId" binding:"required"`
	Amount     int    `json:"amount" binding:"required,gt=0"`
	Note       string `json:"note"`
}

func createTransfer(c *gin.Context) {
	var req transferCreateReq
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.FromUserID == req.ToUserID {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "cannot transfer to self"})
		return
	}
	db := store.GetDB()
	idem := uuid.New().String()
	now := time.Now()
	err := db.Transaction(func(tx *gorm.DB) error {
		var from models.User
		if err := tx.First(&from, req.FromUserID).Error; err != nil {
			return fmt.Errorf("from user: %w", err)
		}
		var to models.User
		if err := tx.First(&to, req.ToUserID).Error; err != nil {
			return fmt.Errorf("to user: %w", err)
		}
		if from.RemainingPoints < req.Amount {
			return fmt.Errorf("insufficient points")
		}
		tr := models.Transfer{
			FromUserID:     req.FromUserID,
			ToUserID:       req.ToUserID,
			Amount:         req.Amount,
			Status:         "processing",
			Note:           &req.Note,
			IdempotencyKey: idem,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		if err := tx.Create(&tr).Error; err != nil {
			return err
		}
		from.RemainingPoints -= req.Amount
		to.RemainingPoints += req.Amount
		if err := tx.Save(&from).Error; err != nil {
			return err
		}
		if err := tx.Save(&to).Error; err != nil {
			return err
		}
		fromLedger := models.PointLedger{
			UserID:       from.ID,
			Change:       -req.Amount,
			BalanceAfter: from.RemainingPoints,
			EventType:    "transfer_out",
			TransferID:   &tr.ID,
			CreatedAt:    now,
		}
		toLedger := models.PointLedger{
			UserID:       to.ID,
			Change:       req.Amount,
			BalanceAfter: to.RemainingPoints,
			EventType:    "transfer_in",
			TransferID:   &tr.ID,
			CreatedAt:    now,
		}
		if err := tx.Create(&fromLedger).Error; err != nil {
			return err
		}
		if err := tx.Create(&toLedger).Error; err != nil {
			return err
		}
		completed := time.Now()
		tr.Status = "completed"
		tr.CompletedAt = &completed
		tr.UpdatedAt = completed
		if err := tx.Save(&tr).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		if err.Error() == "insufficient points" {
			c.JSON(http.StatusConflict, gin.H{"error": "insufficient points"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var tr models.Transfer
	if err := db.Where("idempotency_key = ?", idem).First(&tr).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Header("Idempotency-Key", tr.IdempotencyKey)
	c.JSON(http.StatusCreated, gin.H{"transfer": tr})
}

func listTransfers(c *gin.Context) {
	userId := c.Query("userId")
	page := 1
	pageSize := 20
	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if ps := c.Query("pageSize"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 && v <= 200 {
			pageSize = v
		}
	}
	var transfers []models.Transfer
	db := store.GetDB()
	q := db.Order("created_at desc")
	if userId != "" {
		q = q.Where("from_user_id = ? OR to_user_id = ?", userId, userId)
	}
	var total int64
	q.Model(&models.Transfer{}).Count(&total)
	if err := q.Offset((page - 1) * pageSize).Limit(pageSize).Find(&transfers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": transfers, "page": page, "pageSize": pageSize, "total": total})
}

func getTransfer(c *gin.Context) {
	id := c.Param("id")
	var tr models.Transfer
	db := store.GetDB()
	if err := db.Where("idempotency_key = ?", id).First(&tr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"transfer": tr})
}
