package device

type BindDeviceRequest struct {
	DeviceID string `json:"device_id"`
}

type BindDeviceResponse struct {
	Success bool   `json:"success"`           // 是否成功
	Result  string `json:"result,omitempty"`  // 分析结果（成功时）
	Message string `json:"message,omitempty"` // 错误信息（失败时）
}
