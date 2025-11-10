package models

import "time"

type User struct {
	ID               uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	MemberCode       string `json:"member_code" gorm:"uniqueIndex;not null"`
	MembershipLevel  string `json:"membership_level"`
	Name             string `json:"name"`
	Surname          string `json:"surname"`
	Phone            string `json:"phone"`
	Email            string `json:"email"`
	RegistrationDate string `json:"registration_date"`
	RemainingPoints  int    `json:"remaining_points"`
}

type Transfer struct {
	ID             uint       `json:"transferId" gorm:"primaryKey;autoIncrement"`
	FromUserID     uint       `json:"fromUserId" gorm:"index;not null"`
	ToUserID       uint       `json:"toUserId" gorm:"index;not null"`
	Amount         int        `json:"amount" gorm:"not null"`
	Status         string     `json:"status" gorm:"not null"`
	Note           *string    `json:"note"`
	IdempotencyKey string     `json:"idemKey" gorm:"uniqueIndex;not null"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	CompletedAt    *time.Time `json:"completedAt"`
	FailReason     *string    `json:"failReason"`
}

type PointLedger struct {
	ID           uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID       uint      `json:"userId" gorm:"index;not null"`
	Change       int       `json:"change" gorm:"not null"`
	BalanceAfter int       `json:"balance_after" gorm:"not null"`
	EventType    string    `json:"event_type" gorm:"not null"`
	TransferID   *uint     `json:"transfer_id" gorm:"index"`
	Reference    *string   `json:"reference"`
	Metadata     *string   `json:"metadata"`
	CreatedAt    time.Time `json:"created_at"`
}
