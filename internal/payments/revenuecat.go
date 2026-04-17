package payments

import "strings"

type RevenueCatWebhook struct {
	Event RevenueCatEvent `json:"event"`
}

type RevenueCatEvent struct {
	Type               string `json:"type"`
	AppUserID          string `json:"app_user_id"`
	ProductID          string `json:"product_id"`
	EntitlementID      string `json:"entitlement_id"`
	ExpirationAtMillis *int64 `json:"expiration_at_ms"`
}

func CreditsForProduct(productID string) int {
	switch strings.ToLower(productID) {
	case "credits_50", "dedikodu_credits_50":
		return 50
	case "credits_120", "dedikodu_credits_120":
		return 120
	case "credits_300", "dedikodu_credits_300":
		return 300
	default:
		return 0
	}
}

func IsVIPEvent(eventType string) bool {
	switch strings.ToUpper(eventType) {
	case "INITIAL_PURCHASE", "RENEWAL", "UNCANCELLATION":
		return true
	default:
		return false
	}
}

func IsVIPCancelEvent(eventType string) bool {
	switch strings.ToUpper(eventType) {
	case "CANCELLATION", "EXPIRATION", "BILLING_ISSUE":
		return true
	default:
		return false
	}
}
