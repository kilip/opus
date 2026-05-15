package model

type ApiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ApiResponse struct {
	Success bool      `json:"success"`
	Data    any       `json:"data"`
	Error   *ApiError `json:"error"`
}
