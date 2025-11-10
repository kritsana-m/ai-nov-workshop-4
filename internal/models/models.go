package models

import "time"

// User represents an application user with a points balance.
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

// Transfer represents a points transfer between two users. The IdempotencyKey
// is used to ensure duplicate submissions are detected.
type Transfer struct {
	ID             uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	FromUserID     uint       `json:"from_user_id" gorm:"index;not null"`
	ToUserID       uint       `json:"to_user_id" gorm:"index;not null"`
	Amount         int        `json:"amount" gorm:"not null"`
	Status         string     `json:"status" gorm:"not null"`
	Note           *string    `json:"note"`
	IdempotencyKey string     `json:"idempotency_key" gorm:"uniqueIndex;not null"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	CompletedAt    *time.Time `json:"completed_at"`
	FailReason     *string    `json:"fail_reason"`
}

// PointLedger records point changes for a user, optionally linked to a transfer.
type PointLedger struct {
	ID           uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID       uint      `json:"user_id" gorm:"index;not null"`
	Change       int       `json:"change" gorm:"not null"`
	BalanceAfter int       `json:"balance_after" gorm:"not null"`
	EventType    string    `json:"event_type" gorm:"not null"`
	TransferID   *uint     `json:"transfer_id" gorm:"index"`
	Reference    *string   `json:"reference"`
	Metadata     *string   `json:"metadata"`
	CreatedAt    time.Time `json:"created_at"`
}
