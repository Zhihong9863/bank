package gapi

import (
	"context"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

const (
	grpcGatewayUserAgentHeader = "grpcgateway-user-agent"
	userAgentHeader            = "user-agent"
	xForwardedForHeader        = "x-forwarded-for"
)

type Metadata struct {
	UserAgent string
	ClientIP  string
}

/*
从 gRPC 请求的上下文（context）中提取元数据（metadata）。

元数据（Metadata）在 gRPC 中是键值对的集合，通常用于传递诸如认证令牌、
请求 ID 或其他跨不同服务调用应保持一致的信息。在 gRPC-Gateway 中，
某些 HTTP 请求头部会被自动转换为 gRPC 的元数据并通过上下文传递。

这段代码的目的是创建一个 Metadata 结构体实例，
并填充它与当前请求相关的 User-Agent 和 Client IP 信息。
这对于日志记录、监控或安全目的特别有用，因为我们可以了解是谁在从何处调用您的服务。
*/
func (server *Server) extractMetadata(ctx context.Context) *Metadata {
	mtdt := &Metadata{}

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		/*
			User-Agent：这是一个标准的HTTP请求头部字段，用于指示发起请求的客户端
			（如浏览器或其他网络客户端）的类型和版本。在这段代码中，
			它首先尝试从 grpcGatewayUserAgentHeader 中获取 User-Agent，
			如果没有找到，它会尝试从 userAgentHeader 中获取。
		*/
		if userAgents := md.Get(grpcGatewayUserAgentHeader); len(userAgents) > 0 {
			mtdt.UserAgent = userAgents[0]
		}

		if userAgents := md.Get(userAgentHeader); len(userAgents) > 0 {
			mtdt.UserAgent = userAgents[0]
		}

		/*
			Client IP：xForwardedForHeader 通常用于识别发起请求的原始客户端的 IP 地址，
			特别是当请求通过代理或负载均衡器时。如果这个头部不存在，
			代码会尝试从 peer 信息中获取连接的 IP 地址。
			peer.FromContext 提供了与请求直接相关的网络对等信息，
			例如客户端的 IP 地址和端口。
		*/
		if clientIPs := md.Get(xForwardedForHeader); len(clientIPs) > 0 {
			mtdt.ClientIP = clientIPs[0]
		}
	}

	if p, ok := peer.FromContext(ctx); ok {
		mtdt.ClientIP = p.Addr.String()
	}

	return mtdt
}
