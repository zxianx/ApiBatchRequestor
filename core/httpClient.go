package core

import (
    "fmt"
    "net"
    "net/http"
    "time"
)

// HttpClientReuseMode  HTTP 连接复用模式
const (
    // HttpClientReuseModeShared 默认: 所有 worker 共用一个全局 client（共享连接池）
    HttpClientReuseModeShared = 0
    // HttpClientReuseModePerWorker 每个 worker 协程独立 1 个 client（各自独立连接池）
    HttpClientReuseModePerWorker = 1
    // HttpClientReuseModeShortConn 短连接, 每次请求后关闭 TCP 连接（DisableKeepAlives）
    HttpClientReuseModeShortConn = 2
)

// 模式 1 (PerWorker) 下，每个 client 只服务 1 个 worker，同时只有 1 个 in-flight 请求，
// 连接池不需要大；以下为单 worker 独占 client 的默认连接池配置。
const (
    perWorkerMaxIdleConns        = 8
    perWorkerMaxIdleConnsPerHost = 8
    perWorkerMaxConnsPerHost     = 8
    perWorkerIdleConnTimeout     = 90 * time.Second
)

// DefaultHttpClient 全局共享的 HTTP client（对应模式 0/HttpClientReuseModeShared）。
// 保留为可导出变量以兼容 main.go 中对 Timeout/Proxy 的运行时改写。
var DefaultHttpClient = http.Client{
    Transport: &http.Transport{
        DialContext: (&net.Dialer{
            Timeout:   30 * time.Second,
            KeepAlive: 30 * time.Second,
        }).DialContext,
        ForceAttemptHTTP2:     true,
        MaxIdleConns:          2000,
        MaxIdleConnsPerHost:   1000,
        MaxConnsPerHost:       1000,
        TLSHandshakeTimeout:   10 * time.Second,
        IdleConnTimeout:       30 * time.Second,
        ResponseHeaderTimeout: 60 * time.Second,
        ExpectContinueTimeout: 10 * time.Second,
    },
}

// getClients 按 HttpClientReuseMode 准备 worker 用的 client。
//   模式 0: shared=&DefaultHttpClient,  perWorker=nil
//   模式 1: shared=nil,                 perWorker=workerNum 个独立 client
//   模式 2: shared=新建的短连接 client,  perWorker=nil
// 调用方按 shared 是否为 nil 判断, 不为 nil 就所有 worker 共用它。
func getClients(mode, workerNum int) (shared *http.Client, perWorker []*http.Client, err error) {
    switch mode {
    case HttpClientReuseModeShared:
        shared = &DefaultHttpClient
    case HttpClientReuseModePerWorker:
        perWorker = make([]*http.Client, workerNum)
        for i := 0; i < workerNum; i++ {
            perWorker[i] = buildHttpClient(HttpClientReuseModePerWorker)
        }
    case HttpClientReuseModeShortConn:
        shared = buildHttpClient(HttpClientReuseModeShortConn)
    default:
        err = fmt.Errorf("illegal HttpClientReuseMode:%d, want 0/1/2", mode)
    }
    return
}

// buildHttpClient 构造模式 1/2 用的独立 client（不用于模式 0, 模式 0 直接复用 DefaultHttpClient）。
// 新建的 client 从 DefaultHttpClient 继承 Timeout 和已配置的 Proxy。
func buildHttpClient(mode int) *http.Client {
    transport := &http.Transport{
        DialContext: (&net.Dialer{
            Timeout:   30 * time.Second,
            KeepAlive: 30 * time.Second,
        }).DialContext,
        ForceAttemptHTTP2:     true,
        TLSHandshakeTimeout:   10 * time.Second,
        ResponseHeaderTimeout: 60 * time.Second,
        ExpectContinueTimeout: 10 * time.Second,
    }

    switch mode {
    case HttpClientReuseModePerWorker:
        transport.MaxIdleConns = perWorkerMaxIdleConns
        transport.MaxIdleConnsPerHost = perWorkerMaxIdleConnsPerHost
        transport.MaxConnsPerHost = perWorkerMaxConnsPerHost
        transport.IdleConnTimeout = perWorkerIdleConnTimeout
    case HttpClientReuseModeShortConn:
        // 短连接: 关闭 keep-alive, 单次请求后 TCP 即关闭, 连接池字段无意义
        transport.DisableKeepAlives = true
    }

    // 从 DefaultHttpClient 继承 main.go 配置的 Proxy
    if defTr, ok := DefaultHttpClient.Transport.(*http.Transport); ok && defTr.Proxy != nil {
        transport.Proxy = defTr.Proxy
    }

    return &http.Client{
        Transport: transport,
        Timeout:   DefaultHttpClient.Timeout,
    }
}
