package main

import "time"

type User struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"` // never serialize this to JSON
	BalanceKobo  int64  `json:"balance_kobo"`
	CreatedAt    time.Time `json:"created_at"`
}

type Transaction struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"user_id"`
	Type          string    `json:"type"` // deposit, withdraw, transfer_in, transfer_out
	AmountKobo    int64     `json:"amount_kobo"`
	RelatedUserID *int64    `json:"related_user_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}