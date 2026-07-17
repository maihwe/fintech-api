package main

import (
	"encoding/json"
	"net/http"
)

type amountRequest struct {
	AmountKobo int64 `json:"amount_kobo"`
}

type transferRequest struct {
	ToUsername string `json:"to_username"`
	AmountKobo int64  `json:"amount_kobo"`
}

func balanceHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(userIDKey).(int64)

	var balance int64
	err := db.QueryRow("SELECT balance_kobo FROM users WHERE id = ?", userID).Scan(&balance)
	if err != nil {
		http.Error(w, "failed to fetch balance", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int64{"balance_kobo": balance})
}

func depositHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID := r.Context().Value(userIDKey).(int64)

	var req amountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.AmountKobo <= 0 {
		http.Error(w, "invalid amount", http.StatusBadRequest)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	if _, err := tx.Exec("UPDATE users SET balance_kobo = balance_kobo + ? WHERE id = ?", req.AmountKobo, userID); err != nil {
		http.Error(w, "failed to update balance", http.StatusInternalServerError)
		return
	}
	if _, err := tx.Exec("INSERT INTO transactions (user_id, type, amount_kobo) VALUES (?, 'deposit', ?)", userID, req.AmountKobo); err != nil {
		http.Error(w, "failed to record transaction", http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, "failed to commit transaction", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "deposit successful"})
}

func withdrawHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID := r.Context().Value(userIDKey).(int64)

	var req amountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.AmountKobo <= 0 {
		http.Error(w, "invalid amount", http.StatusBadRequest)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var balance int64
	if err := tx.QueryRow("SELECT balance_kobo FROM users WHERE id = ?", userID).Scan(&balance); err != nil {
		http.Error(w, "failed to fetch balance", http.StatusInternalServerError)
		return
	}
	if balance < req.AmountKobo {
		http.Error(w, "insufficient funds", http.StatusUnprocessableEntity)
		return
	}

	if _, err := tx.Exec("UPDATE users SET balance_kobo = balance_kobo - ? WHERE id = ?", req.AmountKobo, userID); err != nil {
		http.Error(w, "failed to update balance", http.StatusInternalServerError)
		return
	}
	if _, err := tx.Exec("INSERT INTO transactions (user_id, type, amount_kobo) VALUES (?, 'withdraw', ?)", userID, req.AmountKobo); err != nil {
		http.Error(w, "failed to record transaction", http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, "failed to commit transaction", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "withdrawal successful"})
}

func transferHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	fromUserID := r.Context().Value(userIDKey).(int64)

	var req transferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.AmountKobo <= 0 || req.ToUsername == "" {
		http.Error(w, "invalid transfer request", http.StatusBadRequest)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var toUserID int64
	if err := tx.QueryRow("SELECT id FROM users WHERE username = ?", req.ToUsername).Scan(&toUserID); err != nil {
		http.Error(w, "recipient not found", http.StatusNotFound)
		return
	}
	if toUserID == fromUserID {
		http.Error(w, "cannot transfer to yourself", http.StatusBadRequest)
		return
	}

	var balance int64
	if err := tx.QueryRow("SELECT balance_kobo FROM users WHERE id = ?", fromUserID).Scan(&balance); err != nil {
		http.Error(w, "failed to fetch balance", http.StatusInternalServerError)
		return
	}
	if balance < req.AmountKobo {
		http.Error(w, "insufficient funds", http.StatusUnprocessableEntity)
		return
	}

	if _, err := tx.Exec("UPDATE users SET balance_kobo = balance_kobo - ? WHERE id = ?", req.AmountKobo, fromUserID); err != nil {
		http.Error(w, "failed to debit sender", http.StatusInternalServerError)
		return
	}
	if _, err := tx.Exec("UPDATE users SET balance_kobo = balance_kobo + ? WHERE id = ?", req.AmountKobo, toUserID); err != nil {
		http.Error(w, "failed to credit recipient", http.StatusInternalServerError)
		return
	}
	if _, err := tx.Exec("INSERT INTO transactions (user_id, type, amount_kobo, related_user_id) VALUES (?, 'transfer_out', ?, ?)", fromUserID, req.AmountKobo, toUserID); err != nil {
		http.Error(w, "failed to record sender transaction", http.StatusInternalServerError)
		return
	}
	if _, err := tx.Exec("INSERT INTO transactions (user_id, type, amount_kobo, related_user_id) VALUES (?, 'transfer_in', ?, ?)", toUserID, req.AmountKobo, fromUserID); err != nil {
		http.Error(w, "failed to record recipient transaction", http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, "failed to commit transfer", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "transfer successful"})
}

func transactionsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(userIDKey).(int64)

	rows, err := db.Query(`SELECT id, user_id, type, amount_kobo, related_user_id, created_at 
		FROM transactions WHERE user_id = ? ORDER BY created_at DESC`, userID)
	if err != nil {
		http.Error(w, "failed to fetch transactions", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	list := []Transaction{}
	for rows.Next() {
		rows.Err()
		var t Transaction
		if err := rows.Scan(&t.ID, &t.UserID, &t.Type, &t.AmountKobo, &t.RelatedUserID, &t.CreatedAt); err != nil {
			continue
		}
		list = append(list, t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}