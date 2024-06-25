package model

type LoginRequest struct {
	Address  string `json:"address"`
	Password string `json:"password"`
}

type AddSubscriptionRequest struct {
	Address string `json:"address"`
}
