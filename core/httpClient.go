package core

import (
    "net"
    "net/http"
    "time"
)

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
