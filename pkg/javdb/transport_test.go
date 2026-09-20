package javdb

import (
	"fmt"
	"io"
	"net/http"
	"testing"
)

func TestClient_Transport(t *testing.T) {
	// 这是一个手动验证测试，用于检查 utls 传输是否正常工作
	// 我们可以尝试请求一个公共的 HTTPS 站点，例如 httpbin.org 或 google.com

	client := NewClient()

	// 测试目标 URL (使用一个稳定的 HTTPS 站点)
	targetURL := "https://www.javdb.com"

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// 使用我们自定义的 Transport 发送请求
	resp, err := client.httpClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("Successfully connected to %s\n", targetURL)
	fmt.Printf("Status Code: %d\n", resp.StatusCode)

	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		t.Errorf("Failed to read body: %v", err)
	}
}
