package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hellobchain/weekly-assistant/internal/models"
)

func gitlabAuthRequest(baseURL, token string, path string) (*http.Request, error) {
	req, err := http.NewRequest("GET", strings.TrimRight(baseURL, "/")+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("PRIVATE-TOKEN", token)
	return req, nil
}

// FetchGitLabCommits 从私有 GitLab 拉取指定项目的 commit 日志
// baseURL: GitLab地址，如 https://gitlab.company.com
// token: Personal Access Token（在 GitLab 设置中生成，选 api 权限）
// branch: 分支名，可选，为空时拉取所有分支
func FetchGitLabCommits(baseURL, token, projectID, branch, email, startDate, endDate string) ([]map[string]interface{}, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	// 1. 验证认证是否有效
	authReq, err := gitlabAuthRequest(baseURL, token, "/api/v4/user")
	if err != nil {
		return nil, fmt.Errorf("创建认证请求失败: %w", err)
	}
	authResp, err := client.Do(authReq)
	if err != nil {
		return nil, fmt.Errorf("连接 GitLab 失败: %w", err)
	}
	authBody, _ := io.ReadAll(authResp.Body)
	authResp.Body.Close()

	if authResp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("认证失败，请在 GitLab 设置中生成 Personal Access Token 并填入密码字段")
	}
	if authResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("认证接口异常 (%d): %s", authResp.StatusCode, string(authBody))
	}

	// 2. 验证项目是否存在
	encodedProject := strings.ReplaceAll(projectID, "/", "%2F")
	projReq, err := gitlabAuthRequest(baseURL, token, fmt.Sprintf("/api/v4/projects/%s", encodedProject))
	if err != nil {
		return nil, fmt.Errorf("创建项目查询请求失败: %w", err)
	}
	projResp, err := client.Do(projReq)
	if err != nil {
		return nil, fmt.Errorf("查询项目失败: %w", err)
	}
	projBody, _ := io.ReadAll(projResp.Body)
	projResp.Body.Close()

	if projResp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("项目 %s 不存在或无访问权限，请检查 project_id", projectID)
	}
	if projResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("查询项目异常 (%d): %s", projResp.StatusCode, string(projBody))
	}

	// 3. 拉取 commits
	params := url.Values{}
	params.Set("since", startDate+"T00:00:00Z")
	params.Set("until", endDate+"T23:59:59Z")
	params.Set("per_page", "1000")
	if branch != "" {
		params.Set("ref_name", branch)
	}

	u := fmt.Sprintf("%s/api/v4/projects/%s/repository/commits?%s",
		strings.TrimRight(baseURL, "/"), encodedProject, params.Encode())

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("PRIVATE-TOKEN", token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 GitLab 失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitLab 返回状态 %d: %s", resp.StatusCode, string(body))
	}

	var raw []models.GitlabCommitRaw
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	result := make([]map[string]interface{}, 0, len(raw))
	for _, c := range raw {
		msg := c.Message
		if c.Title != "" {
			msg = c.Title
		}
		if email != "" && c.AuthorEmail != email {
			log.Printf("GitLab: 忽略非 %s 而是 %s 的提交 %s\n", email, c.AuthorEmail, c.ID)
			continue
		}
		result = append(result, map[string]interface{}{
			"id":            c.ID,
			"message":       msg,
			"authored_date": c.AuthoredDate,
			"web_url":       c.WebURL,
		})
	}

	log.Printf("GitLab: 从 %s 拉取 %s 的 %d 条 commit (%s ~ %s)", projectID, email, len(result), startDate, endDate)
	return result, nil
}
