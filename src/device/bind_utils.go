package device

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
)

// GenerateBindKey 根据设备ID和服务器密钥生成绑定密钥
// 使用HMAC(device_id, server_secret)算法生成绑定密钥
// @param deviceID 设备唯一标识符
// @param serverSecret 服务器密钥，用作HMAC的密钥
// @return 返回十六进制格式的绑定密钥
func GenerateBindKey(deviceID string, serverSecret string) string {
	// 创建HMAC-SHA256哈希器，使用serverSecret作为密钥
	h := hmac.New(sha256.New, []byte(serverSecret))
	
	// 写入设备ID作为消息
	h.Write([]byte(deviceID))
	
	// 计算HMAC值并返回十六进制字符串
	return fmt.Sprintf("%x", h.Sum(nil))
}

// ValidateBindKey 验证绑定密钥是否正确
// 通过重新计算HMAC并与存储的bind_key进行比较来验证
// @param deviceID 设备唯一标识符
// @param serverSecret 服务器密钥
// @param storedBindKey 存储在数据库中的绑定密钥
// @return 返回验证结果，true表示验证通过
func ValidateBindKey(deviceID string, serverSecret string, storedBindKey string) bool {
	// 重新计算bind_key
	expectedBindKey := GenerateBindKey(deviceID, serverSecret)
	
	// 使用恒定时间比较防止时序攻击
	return hmac.Equal([]byte(expectedBindKey), []byte(storedBindKey))
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
