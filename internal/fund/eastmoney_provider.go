package fund

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	fundCodeListURL = "https://fund.eastmoney.com/js/fundcode_search.js"
	fundPageURL     = "https://fund.eastmoney.com/%s.html"
	fundEstimateURL = "https://fundgz.1234567.com.cn/js/%s.js"
	fundHistoryURL  = "https://fundf10.eastmoney.com/F10DataApi.aspx?type=lsjz&code=%s&page=1&per=400&sdate=%s&edate=%s"
)

type EastMoneyProvider struct {
	client     *http.Client
	cacheMu    sync.RWMutex
	candidates []SearchCandidate
	cachedAt   time.Time
}

func NewEastMoneyProvider() *EastMoneyProvider {
	return &EastMoneyProvider{
		client: &http.Client{Timeout: 12 * time.Second},
	}
}

func (p *EastMoneyProvider) Search(ctx context.Context, query string, limit int) ([]SearchCandidate, error) {
	all, err := p.loadCandidates(ctx)
	if err != nil {
		return nil, err
	}

	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return nil, nil
	}

	results := make([]SearchCandidate, 0, limit)
	for _, item := range all {
		if strings.EqualFold(item.Code, query) {
			return []SearchCandidate{item}, nil
		}
	}

	for _, item := range all {
		if strings.Contains(strings.ToLower(item.Name), query) || strings.Contains(strings.ToLower(item.Code), query) {
			results = append(results, item)
			if len(results) >= limit {
				break
			}
		}
	}

	return results, nil
}

func (p *EastMoneyProvider) GetProfile(ctx context.Context, code string) (*Profile, error) {
	candidates, err := p.Search(ctx, code, 1)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, errors.New("fund not found")
	}

	estimateBody, err := p.fetchText(ctx, fmt.Sprintf(fundEstimateURL, code))
	if err != nil {
		return nil, err
	}
	estimate, err := parseEstimatePayload(estimateBody)
	if err != nil {
		return nil, err
	}

	pageBody, err := p.fetchText(ctx, fmt.Sprintf(fundPageURL, code))
	if err != nil {
		return nil, err
	}

	profile := &Profile{
		Code:         candidates[0].Code,
		Name:         firstNonEmpty(estimate.Name, candidates[0].Name),
		Type:         firstNonEmpty(extractType(pageBody), candidates[0].Type),
		Company:      extractCompany(pageBody),
		LatestNAV:    estimate.UnitNAV,
		DailyChange:  estimate.ChangeRate,
		AsOfDate:     estimate.Date,
		EstimatedNAV: estimate.EstimatedNAV,
	}
	return profile, nil
}

func (p *EastMoneyProvider) GetTrend(ctx context.Context, code string, rangeKey string) (*Trend, error) {
	start, end := rangeDates(rangeKey)
	body, err := p.fetchText(ctx, fmt.Sprintf(fundHistoryURL, code, start, end))
	if err != nil {
		return nil, err
	}

	points := parseHistoryPoints(body)
	if len(points) == 0 {
		return nil, errors.New("trend data not found")
	}

	sort.Slice(points, func(i, j int) bool { return points[i].Date < points[j].Date })
	return &Trend{Range: rangeKey, Points: points}, nil
}

func (p *EastMoneyProvider) loadCandidates(ctx context.Context) ([]SearchCandidate, error) {
	p.cacheMu.RLock()
	if len(p.candidates) > 0 && time.Since(p.cachedAt) < 12*time.Hour {
		defer p.cacheMu.RUnlock()
		return append([]SearchCandidate(nil), p.candidates...), nil
	}
	p.cacheMu.RUnlock()

	p.cacheMu.Lock()
	defer p.cacheMu.Unlock()
	if len(p.candidates) > 0 && time.Since(p.cachedAt) < 12*time.Hour {
		return append([]SearchCandidate(nil), p.candidates...), nil
	}

	body, err := p.fetchText(ctx, fundCodeListURL)
	if err != nil {
		return nil, err
	}
	body = strings.TrimPrefix(body, "\ufeff")
	body = strings.TrimSpace(strings.TrimPrefix(body, "var r = "))
	body = strings.TrimSuffix(body, ";")

	var raw [][]string
	if err := json.Unmarshal([]byte(body), &raw); err != nil {
		return nil, err
	}

	candidates := make([]SearchCandidate, 0, len(raw))
	for _, row := range raw {
		if len(row) < 4 {
			continue
		}
		candidates = append(candidates, SearchCandidate{
			Code: row[0],
			Name: row[2],
			Type: row[3],
		})
	}

	p.candidates = candidates
	p.cachedAt = time.Now()
	return append([]SearchCandidate(nil), p.candidates...), nil
}

func (p *EastMoneyProvider) fetchText(ctx context.Context, target string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("provider request failed: %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

type estimatePayload struct {
	Code         string `json:"fundcode"`
	Name         string `json:"name"`
	Date         string `json:"jzrq"`
	UnitNAV      string `json:"dwjz"`
	EstimatedNAV string `json:"gsz"`
	ChangeRate   string `json:"gszzl"`
}

func parseEstimatePayload(body string) (*estimatePayload, error) {
	body = strings.TrimSpace(body)
	body = strings.TrimPrefix(body, "jsonpgz(")
	body = strings.TrimSuffix(body, ");")

	var payload estimatePayload
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

func extractType(page string) string {
	re := regexp.MustCompile(`类型：.*?>([^<]+)</a>`)
	match := re.FindStringSubmatch(page)
	if len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

func extractCompany(page string) string {
	re := regexp.MustCompile(`管 理 人：.*?>([^<]+)</a>`)
	match := re.FindStringSubmatch(page)
	if len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

func rangeDates(rangeKey string) (string, string) {
	end := time.Now()
	start := end.AddDate(0, 0, -7)
	switch rangeKey {
	case "day":
		start = end.AddDate(0, 0, -7)
	case "month":
		start = end.AddDate(0, -1, 0)
	case "3m":
		start = end.AddDate(0, -3, 0)
	case "6m":
		start = end.AddDate(0, -6, 0)
	case "1y":
		start = end.AddDate(-1, 0, 0)
	default:
		start = end.AddDate(0, -1, 0)
	}
	return start.Format("2006-01-02"), end.Format("2006-01-02")
}

func parseHistoryPoints(body string) []TrendPoint {
	body = strings.ReplaceAll(body, "\n", "")
	re := regexp.MustCompile(`<tr><td>([^<]+)</td><td[^>]*>([^<]+)</td>`)
	matches := re.FindAllStringSubmatch(body, -1)
	points := make([]TrendPoint, 0, len(matches))
	currentYear := time.Now().Year()

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		date := strings.TrimSpace(match[1])
		if len(date) == 5 {
			date = fmt.Sprintf("%d-%s", currentYear, date)
		}
		value, err := strconv.ParseFloat(strings.TrimSpace(match[2]), 64)
		if err != nil {
			continue
		}
		points = append(points, TrendPoint{
			Date:  date,
			Value: value,
		})
	}

	return points
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func normalizeQuery(query string) string {
	value, _ := url.QueryUnescape(strings.TrimSpace(query))
	return value
}
