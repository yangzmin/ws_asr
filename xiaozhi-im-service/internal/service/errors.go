package service

import "errors"

var (
	// 连接相关错误
	ErrConnectionClosed   = errors.New("连接已关闭")
	ErrConnectionNotFound = errors.New("连接未找到")
	ErrSendChannelFull    = errors.New("发送通道已满")

	// gRPC相关错误
	ErrGRPCClientNotReady = errors.New("gRPC客户端未就绪")
	ErrGRPCStreamClosed   = errors.New("gRPC流已关闭")
	ErrGRPCConnectionFail = errors.New("gRPC连接失败")

	// 消息相关错误
	ErrInvalidMessageType   = errors.New("无效的消息类型")
	ErrMessageConvertFailed = errors.New("消息转换失败")
	ErrMessageEncodeFailed  = errors.New("消息编码失败")
	ErrMessageDecodeFailed  = errors.New("消息解码失败")

	// 认证相关错误
	ErrAuthenticationFailed = errors.New("认证失败")
	ErrTokenInvalid         = errors.New("Token无效")
	ErrTokenExpired         = errors.New("Token已过期")

	// 服务相关错误
	ErrServiceNotReady = errors.New("服务未就绪")
	ErrServiceStopped  = errors.New("服务已停止")
)