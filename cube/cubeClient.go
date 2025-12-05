package cube

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

type CubeClient struct {
	Config *Config
	Resty  *resty.Client
}

// UploadResponse 定义 Cube 文件上传 API 的标准响应结构
type UploadResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		ObjectKey string `json:"object_key"`
		URL       string `json:"url"`
		FileID    string `json:"file_id"`
	} `json:"data"`
}

// NewCubeClient 创建一个 CubeClient。可以注入一个已有的 resty.Client（可为 nil）。
// 当 httpClient 为 nil 时，函数会创建并配置一个默认的 resty.Client。
func NewCubeClient(cfg *Config, httpClient *resty.Client) (*CubeClient, error) {
	if cfg == nil {
		return nil, errors.New("cube: cfg 不能为空")
	}
	if cfg.BaseURL == "" {
		return nil, errors.New("cube: BaseURL 未配置")
	}

	// 确保默认超时时间
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}

	timeoutDuration := cfg.Timeout

	var client *resty.Client
	if httpClient != nil {
		client = httpClient
	} else {
		client = resty.New()
	}

	// 应用基础配置到 resty 客户端
	client.SetBaseURL(strings.TrimRight(cfg.BaseURL, "/")).
		SetTimeout(timeoutDuration).
		SetHeader("Key", cfg.APIKey)

	return &CubeClient{Config: cfg, Resty: client}, nil
}

// UploadFile 上传文件到存储立方
func (c *CubeClient) UploadFile(localPath, location string, convertWebp, useUuid bool) (string, error) {
	bucketName := c.Config.DefaultBucketName
	if bucketName == "" {
		return "", errors.New("默认存储桶名称未配置，请检查 CubeConfig")
	}

	file, err := os.Open(localPath)
	if err != nil {
		return "", fmt.Errorf("无法打开本地文件: %w", err)
	}
	defer file.Close()

	fileName := filepath.Base(localPath)

	// 构建表单数据，避免重复调用 SetFormData 覆盖原有 map
	form := map[string]string{
		"bucket":       bucketName,
		"convert_webp": fmt.Sprintf("%v", convertWebp),
		"use_uuid":     fmt.Sprintf("%v", useUuid),
	}
	if location != "" {
		form["location"] = location
	}

	var result UploadResponse
	req := c.Resty.R().
		SetFileReader("file", fileName, file).
		SetFormData(form).
		SetResult(&result)

	resp, err := req.Post("/api/upload")

	if err != nil {
		return "", fmt.Errorf("cube 客户端请求失败: %w", err)
	}

	if resp.IsError() {
		return "", fmt.Errorf("cube HTTP 错误: %s (Status: %d). Response Body: %s", resp.Status(), resp.StatusCode(), resp.String())
	}

	// 检查是否有 JSON 解析错误（仅当 HTTP 状态码为成功时）
	if resp.IsSuccess() && resp.Error() != nil {
		// resp.Error() 返回一个错误对象，指示 SetResult() 失败
		return "", fmt.Errorf("cube 响应体JSON解析失败: %v. Response Body: %s",
			resp.Error(), resp.String())
	}

	if result.Code != 200 {
		return "", NewCubeError(result.Code, result.Msg)
	}

	objKey := result.Data.ObjectKey
	if objKey == "" {
		return "", NewCubeError(200500, "上传成功但 ObjectKey 缺失")
	}
	return objKey, nil
}

// DeleteFile 删除文件
func (c *CubeClient) DeleteFile(objectKey string) error {
	bucketName := c.Config.DefaultBucketName
	if bucketName == "" {
		return errors.New("默认存储桶名称未配置，请检查 CubeConfig")
	}

	var result UploadResponse
	resp, err := c.Resty.R().
		SetQueryParams(map[string]string{
			"bucket":     bucketName,
			"object_key": objectKey,
		}).
		SetResult(&result).
		Delete("/api/delete")
	if err != nil {
		return fmt.Errorf("cube 客户端删除请求失败: %w", err)
	}

	if resp.IsError() {
		return fmt.Errorf("cube HTTP 错误: %s (Status: %d)", resp.Status(), resp.StatusCode())
	}

	if result.Code != 200 {
		return NewCubeError(result.Code, result.Msg)
	}

	return nil
}

// GetFileURL 返回文件访问 URL（会对 objectKey 做 URL 编码）
func (c *CubeClient) GetFileURL(objectKey string, thumbnail bool) string {
	encodedKey := url.QueryEscape(objectKey)
	baseURL := strings.TrimRight(c.Config.BaseURL, "/")
	finalURL := fmt.Sprintf("%s/api/file?bucket=%s&object_key=%s", 
					baseURL, 
					c.Config.DefaultBucketName,
					encodedKey)
	if thumbnail {
		finalURL += "&thumbnail=true"
	}
	return finalURL
}
