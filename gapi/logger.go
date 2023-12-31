package gapi

import (
	"context"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

//logger.go 文件中定义了两个用于记录日志的函数，一个针对 gRPC 请求，另一个针对 HTTP 请求。

/*
这是一个 gRPC 中间件，用于在处理 gRPC 请求时记录日志。它会包装实际的处理函数，在请求处理前后记录日志。

记录开始时间：在请求处理开始时记录当前时间。
处理请求：调用实际的请求处理函数。
计算持续时间：计算处理请求所用的时间。
获取状态码：从错误中提取 gRPC 状态码。
构建日志：基于请求的结果构建日志条目。如果请求成功，使用信息级别日志；如果有错误，使用错误级别并记录错误。
记录日志：记录请求的方法、状态码、文本描述和持续时间。
返回结果：返回处理函数的结果和错误。
*/
func GrpcLogger(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (resp interface{}, err error) {
	startTime := time.Now()
	result, err := handler(ctx, req)
	duration := time.Since(startTime)

	statusCode := codes.Unknown
	if st, ok := status.FromError(err); ok {
		statusCode = st.Code()
	}

	logger := log.Info()
	if err != nil {
		logger = log.Error().Err(err)
	}

	logger.Str("protocol", "grpc").
		Str("method", info.FullMethod).
		Int("status_code", int(statusCode)).
		Str("status_text", statusCode.String()).
		Dur("duration", duration).
		Msg("received a gRPC request")

	return result, err
}

// 这是一个 HTTP 响应记录器，用于拦截和记录 HTTP 响应的状态码和正文内容。
type ResponseRecorder struct {
	http.ResponseWriter
	StatusCode int
	Body       []byte
}

// 拦截状态码和正文内容：通过覆写 WriteHeader 和 Write 方法来记录响应的状态码和正文内容。
func (rec *ResponseRecorder) WriteHeader(statusCode int) {
	rec.StatusCode = statusCode
	rec.ResponseWriter.WriteHeader(statusCode)
}

func (rec *ResponseRecorder) Write(body []byte) (int, error) {
	rec.Body = body
	return rec.ResponseWriter.Write(body)
}

/*
这是一个 HTTP 中间件，用于在处理 HTTP 请求时记录日志。

记录开始时间：在请求处理开始时记录当前时间。
处理请求：使用 ResponseRecorder 来处理请求，从而可以记录响应的状态码和正文。
计算持续时间：计算处理请求所用的时间。
构建日志：基于请求的结果构建日志条目。如果响应状态码不是200 OK，则使用错误级别日志并记录正文内容。
记录日志：记录请求的协议、方法、路径、状态码、文本描述和持续时间。
*/
func HttpLogger(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		startTime := time.Now()
		rec := &ResponseRecorder{
			ResponseWriter: res,
			StatusCode:     http.StatusOK,
		}
		handler.ServeHTTP(rec, req)
		duration := time.Since(startTime)

		logger := log.Info()
		if rec.StatusCode != http.StatusOK {
			logger = log.Error().Bytes("body", rec.Body)
		}

		logger.Str("protocol", "http").
			Str("method", req.Method).
			Str("path", req.RequestURI).
			Int("status_code", rec.StatusCode).
			Str("status_text", http.StatusText(rec.StatusCode)).
			Dur("duration", duration).
			Msg("received a HTTP request")
	})
}
