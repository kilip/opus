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

// NewSuccessResponse creates a successful API response
func NewSuccessResponse(data any) ApiResponse {
	return ApiResponse{
		Success: true,
		Data:    data,
	}
}

// NewErrorResponse creates an error API response
func NewErrorResponse(code, message string) ApiResponse {
	return ApiResponse{
		Success: false,
		Error: &ApiError{
			Code:    code,
			Message: message,
		},
	}
}
