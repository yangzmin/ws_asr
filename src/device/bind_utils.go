package device

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"time"
)

// GenerateBindKey 根据设备ID和服务器密钥生成绑定密钥
// 使用HMAC(device_id, server_secret)算法生成绑定密钥
// @param deviceID 设备唯一标识符
// @param serverSecret 服务器密钥，用作HMAC的密钥
// @return 返回十六进制格式的绑定密钥
func GenerateBindKey(deviceID string, serverSecret string, userID uint) string {
	timestamp := time.Now().Unix()
	h := hmac.New(sha256.New, []byte(serverSecret))

	// 组合：设备ID + 用户ID + 创建时间戳
	data := fmt.Sprintf("%s:%d:%d", deviceID, userID, timestamp)
	h.Write([]byte(data))

	return fmt.Sprintf("%x", h.Sum(nil))
}

// ValidateDeviceID 验证设备ID格式
func ValidateDeviceID(deviceID string) bool {
	if deviceID == "" {
		return false
	}

	// 设备ID长度限制（1-64字符）
	if len(deviceID) < 1 || len(deviceID) > 64 {
		return false
	}

	return true
}
