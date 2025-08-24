package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/dxy0427/emby-302/config"
)

var authTokenRegex = regexp.MustCompile(`(?i)token="([^"]+)"`)

type PlaybackInfoResponse struct {
	MediaSources []struct {
		Path string `json:"Path"`
	} `json:"MediaSources"`
}

type AppState struct {
	Config *config.Config
	Client *http.Client
	Proxy  *httputil.ReverseProxy
}

func NewAppState(cfg *config.Config) (*AppState, error) {
	embyURL, err := url.Parse(cfg.Emby.Host)
	if err != nil {
		return nil, fmt.Errorf("无法解析 Emby Host: %w", err)
	}
	proxy := httputil.NewSingleHostReverseProxy(embyURL)
	return &AppState{
		Config: cfg,
		Client: &http.Client{Timeout: 10 * time.Second},
		Proxy:  proxy,
	}, nil
}

func extractAPIKey(r *http.Request, configAPIKey string) string {
	if apiKey := r.URL.Query().Get("api_key"); apiKey != "" {
		return apiKey
	}
	if apiKey := r.URL.Query().Get("X-Emby-Token"); apiKey != "" {
		return apiKey
	}
	if authHeader := r.Header.Get("X-Emby-Authorization"); authHeader != "" {
		matches := authTokenRegex.FindStringSubmatch(authHeader)
		if len(matches) > 1 {
			return matches[1]
		}
	}
	return configAPIKey
}

func (app *AppState) getEmbyMediaPath(ctx context.Context, itemID, mediaSourceID, apiKey string) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("API Key 为空，无法请求 Emby")
	}
	apiURL := fmt.Sprintf("%s/emby/Items/%s/PlaybackInfo?MediaSourceId=%s&api_key=%s",
		strings.TrimSuffix(app.Config.Emby.Host, "/"), itemID, mediaSourceID, apiKey)
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}
	resp, err := app.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求 Emby API 失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return "", fmt.Errorf("Emby API 鉴权失败 (401 Unauthorized)，请检查 API Key 是否有效")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Emby API 返回非 200 状态码: %d", resp.StatusCode)
	}
	var playbackInfo PlaybackInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&playbackInfo); err != nil {
		return "", fmt.Errorf("解析 Emby API 响应失败: %w", err)
	}
	if len(playbackInfo.MediaSources) == 0 {
		return "", fmt.Errorf("未找到媒体源 (MediaSources)")
	}
	return playbackInfo.MediaSources[0].Path, nil
}

func (app *AppState) RootHandler(w http.ResponseWriter, r *http.Request) {
	userAgent := r.Header.Get("User-Agent")
	if app.Config.ClientFilter.ShouldBlockRequest(userAgent) {
		log.Printf("[INFO] 客户端 User-Agent: '%s'。根据防火墙规则，已禁止其访问。", userAgent)
		http.Error(w, "此客户端已被禁止访问", http.StatusForbidden)
		return
	}

	if app.Config.Emby.DownloadStrategy == "403" && (strings.Contains(r.URL.Path, "/Items/") && strings.Contains(r.URL.Path, "/Download")) {
		log.Printf("[INFO] 已拦截下载请求: %s", r.URL.Path)
		http.Error(w, "下载已被策略禁止", http.StatusForbidden)
		return
	}

	isPlayRequest := strings.Contains(r.URL.Path, "/Videos/") || strings.Contains(r.URL.Path, "/videos/")
	if isPlayRequest {
		params := r.URL.Query()
		mediaSourceID := params.Get("MediaSourceId")
		if mediaSourceID == "" {
			mediaSourceID = params.Get("mediaSourceId")
		}

		if mediaSourceID != "" {
			pathParts := strings.Split(r.URL.Path, "/")
			itemID := pathParts[len(pathParts)-2]
			apiKey := extractAPIKey(r, app.Config.Emby.APIKey)

			mediaPath, err := app.getEmbyMediaPath(r.Context(), itemID, mediaSourceID, apiKey)
			if err != nil {
				log.Printf("[ERROR] 获取 Emby 媒体路径失败: %v", err)
				http.Error(w, fmt.Sprintf("无法获取媒体信息: %v", err), http.StatusInternalServerError)
				return
			}
			log.Printf("[INFO] 获取到 Emby 原始路径: %s", mediaPath)

			finalURL := app.Config.Emby.Strm.MapPath(mediaPath)

			if finalURL != mediaPath {
				log.Printf("[INFO] 规则匹配成功, 重定向到: %s", finalURL)
				http.Redirect(w, r, finalURL, http.StatusFound)
				return
			}
		}
	}

	app.Proxy.ServeHTTP(w, r)
}