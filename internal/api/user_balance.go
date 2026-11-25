package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/LightningTipBot/LightningTipBot/internal/lnbits"
	"github.com/LightningTipBot/LightningTipBot/internal/telegram"
	log "github.com/sirupsen/logrus"
)

// UserBalanceRequest represents the JSON request for the user balance API
type UserBalanceRequest struct {
	TelegramID int64 `json:"telegram_id"` // Telegram user ID
}

// PotBalance is one savings pot in the user balance response.
type PotBalance struct {
	Name    string `json:"name"`
	Balance int64  `json:"balance"` // sats
}

// UserBalanceResponse represents the JSON response for the user balance API.
//
// Savings pots are not separate wallets: their sats stay in the user's LNbits
// wallet and are only earmarked in the bot's DB. So Balance is the whole
// wallet (pots included), AvailableBalance is what the user can spend
// (wallet minus pots minus sats held by in-flight payments, never below 0; the
// same figure ReserveBalance checks a send against) and PotBalance is the sum
// of the pots.
type UserBalanceResponse struct {
	Success             bool         `json:"success"`
	TelegramID          int64        `json:"telegram_id"`
	Balance             int64        `json:"balance"`               // Whole wallet in satoshis, pots included
	BalanceLKR          string       `json:"balance_lkr,omitempty"` // LKR conversion
	AvailableBalance    int64        `json:"available_balance"`     // Spendable sats (wallet minus pots and in-flight payments)
	AvailableBalanceLKR string       `json:"available_balance_lkr,omitempty"`
	PotBalance          int64        `json:"pot_balance"` // Sats locked in savings pots
	PotBalanceLKR       string       `json:"pot_balance_lkr,omitempty"`
	Pots                []PotBalance `json:"pots"` // Each pot, oldest first; [] when none
	FirstName           string       `json:"first_name,omitempty"`
	LastName            string       `json:"last_name,omitempty"`
	Username            string       `json:"username,omitempty"`
	CreatedAt           string       `json:"created_at,omitempty"`
	Message             string       `json:"message,omitempty"`
	// Error repeats Message on a 400 (bad request): those answered {"error": ...}
	// before, and callers read that field.
	Error string `json:"error,omitempty"`
}

// UserBalance handles the /api/v1/userbalance endpoint for getting user balance by Telegram ID
func (s Service) UserBalance(w http.ResponseWriter, r *http.Request) {
	var req UserBalanceRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Errorf("[api/userbalance] Invalid JSON request: %v", err)
		writeBalanceError(w, http.StatusBadRequest, 0, "Invalid JSON request")
		return
	}

	// Validate request
	if req.TelegramID <= 0 {
		writeBalanceError(w, http.StatusBadRequest, req.TelegramID, "Invalid Telegram ID")
		return
	}

	// Get user by Telegram ID
	user, err := telegram.GetUserByTelegramID(req.TelegramID, *s.Bot)
	if err != nil {
		log.Errorf("[api/userbalance] Failed to get user by Telegram ID %d: %v", req.TelegramID, err)
		writeBalanceFailure(w, req.TelegramID, "User not found or has no wallet")
		return
	}

	// Get user balance
	balance, err := s.Bot.GetUserBalance(user)
	if err != nil {
		log.Errorf("[api/userbalance] Failed to get balance for user %d: %v", req.TelegramID, err)
		writeBalanceFailure(w, req.TelegramID, "Failed to retrieve balance")
		return
	}

	// Pots are earmarked in the bot's DB; without them we can't say what is
	// spendable, so fail rather than report the whole wallet as available.
	pots, err := s.Bot.ListUserPots(user)
	if err != nil {
		log.Errorf("[api/userbalance] Failed to get pots for user %d: %v", req.TelegramID, err)
		writeBalanceFailure(w, req.TelegramID, "Failed to retrieve balance")
		return
	}
	// Sats held by this user's payments that lnbits hasn't accepted yet, so
	// available_balance is what ReserveBalance would let a send use now. The
	// wallet and the pots are separate reads, so a pot move in between can make
	// one answer off by that amount; fine for a display figure.
	reserved := telegram.ReservedBalance(req.TelegramID)
	available, potTotal, potList := balanceBreakdown(balance, pots, reserved)

	// Convert to LKR if possible
	balanceLKR := getLKRValue(balance)

	// Extract user details
	var firstName, lastName, username string
	if user.Telegram != nil {
		firstName = user.Telegram.FirstName
		lastName = user.Telegram.LastName
		username = user.Telegram.Username
	}

	response := UserBalanceResponse{
		Success:             true,
		TelegramID:          req.TelegramID,
		Balance:             balance,
		BalanceLKR:          balanceLKR,
		AvailableBalance:    available,
		AvailableBalanceLKR: getLKRValue(available),
		PotBalance:          potTotal,
		PotBalanceLKR:       getLKRValue(potTotal),
		Pots:                potList,
		FirstName:           firstName,
		LastName:            lastName,
		Username:            username,
		CreatedAt:           user.CreatedAt.Format(time.RFC3339),
		Message:             "Balance retrieved successfully",
	}

	log.Infof("[api/userbalance] Balance retrieved for user %d: %d sats (available %d, pots %d)", req.TelegramID, balance, available, potTotal)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// balanceBreakdown splits a wallet balance into spendable sats and sats held
// in pots. Spendable is the wallet minus the pots minus reserved (sats held by
// in-flight payments), never below 0: what ReserveBalance checks a send
// against. The pot list is never nil, so it encodes as [].
func balanceBreakdown(walletBalance int64, pots []lnbits.SavingsPot, reserved int64) (available, potTotal int64, list []PotBalance) {
	list = make([]PotBalance, 0, len(pots))
	for _, p := range pots {
		potTotal += p.Balance
		list = append(list, PotBalance{Name: p.Name, Balance: p.Balance})
	}
	available = walletBalance - potTotal - reserved
	if available < 0 {
		available = 0
	}
	return available, potTotal, list
}

// balanceFailure is the answer when the balance can't be read. It has the
// same shape as a success (pots is [], not null), so a client that reads the
// fields before checking success still gets an empty list.
func balanceFailure(telegramID int64, message string) UserBalanceResponse {
	return UserBalanceResponse{
		Success:    false,
		TelegramID: telegramID,
		Pots:       []PotBalance{},
		Message:    message,
	}
}

// writeBalanceFailure answers 200 with success false: the request was fine,
// the balance couldn't be read.
func writeBalanceFailure(w http.ResponseWriter, telegramID int64, message string) {
	writeBalanceError(w, http.StatusOK, telegramID, message)
}

// writeBalanceError writes the failure shape with status. A 400 also sets
// error, as RespondError did, so a caller reading that field still finds the
// message.
func writeBalanceError(w http.ResponseWriter, status int, telegramID int64, message string) {
	resp := balanceFailure(telegramID, message)
	if status == http.StatusBadRequest {
		resp.Error = message
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}
