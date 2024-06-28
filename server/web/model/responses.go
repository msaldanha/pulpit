package model

type BaseResponse struct {
	BasePath string `json:"base_path"`
}

type LoginResponse struct {
	Token string `json:"token"`
}
