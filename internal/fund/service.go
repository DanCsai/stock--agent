package fund

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

type Service struct {
	provider Provider
}

func NewService(provider Provider) *Service {
	return &Service{provider: provider}
}

func (s *Service) ResolveQuery(ctx context.Context, query string, rangeKey string) (*QueryResult, error) {
	query = normalizeQuery(query)
	if query == "" {
		return nil, nil
	}
	rangeKey = NormalizeRangeKey(rangeKey)

	code := extractCode(query)
	if code != "" {
		return s.lookupByCode(ctx, code, rangeKey)
	}

	var candidates []SearchCandidate
	var err error
	for _, variant := range queryVariants(query) {
		candidates, err = s.provider.Search(ctx, variant, 5)
		if err != nil {
			return nil, err
		}
		if len(candidates) > 0 {
			query = variant
			break
		}
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	if len(candidates) == 1 {
		return s.lookupByCode(ctx, candidates[0].Code, rangeKey)
	}

	return &QueryResult{
		Mode:       "candidates",
		Query:      query,
		Candidates: candidates,
	}, nil
}

func (s *Service) LookupFund(ctx context.Context, code string, rangeKey string) (*QueryResult, error) {
	return s.lookupByCode(ctx, code, NormalizeRangeKey(rangeKey))
}

func (s *Service) lookupByCode(ctx context.Context, code string, rangeKey string) (*QueryResult, error) {
	profile, err := s.provider.GetProfile(ctx, code)
	if err != nil {
		return nil, err
	}
	trend, err := s.provider.GetTrend(ctx, code, rangeKey)
	if err != nil {
		return nil, err
	}

	return &QueryResult{
		Mode:                      "detail",
		Query:                     code,
		Profile:                   profile,
		Trend:                     trend,
		RecommendationPlaceholder: "推荐模块占位：后续将由 Agent 综合基金特征、走势与策略偏好给出建议。",
		AnalysisPlaceholder:       "分析模块占位：后续将支持对单只基金、多只基金对比和推荐理由进行智能分析。",
	}, nil
}

func extractCode(query string) string {
	re := regexp.MustCompile(`\b\d{6}\b`)
	match := re.FindString(query)
	return strings.TrimSpace(match)
}

func NormalizeRangeKey(rangeKey string) string {
	switch strings.TrimSpace(rangeKey) {
	case "day", "month", "3m", "6m", "1y":
		return rangeKey
	default:
		return "month"
	}
}

func ShouldAttemptQuery(query string) bool {
	query = normalizeQuery(query)
	if query == "" {
		return false
	}
	if extractCode(query) != "" {
		return true
	}

	keywords := []string{"基金", "净值", "走势", "涨跌", "代码", "搜索", "查询", "分析", "推荐"}
	for _, keyword := range keywords {
		if strings.Contains(query, keyword) {
			return true
		}
	}

	query = strings.TrimSpace(strings.NewReplacer(" ", "", "　", "").Replace(query))
	length := len([]rune(query))
	return length >= 2 && length <= 12
}

func BuildSummary(result *QueryResult) string {
	if result == nil {
		return ""
	}

	if result.Mode == "candidates" {
		return fmt.Sprintf("我找到了 %d 只和“%s”相关的基金，你可以先从候选里挑一只查看详情。", len(result.Candidates), result.Query)
	}

	if result.Profile == nil {
		return ""
	}

	name := firstNonEmpty(result.Profile.Name, result.Profile.Code)
	changeText := strings.TrimSpace(result.Profile.DailyChange)
	if changeText == "" {
		return fmt.Sprintf("已为你查到 %s 的基础信息和近期走势。", name)
	}

	return fmt.Sprintf("已为你查到 %s 的基础信息和近期走势，当前估算涨跌为 %s。", name, changeText)
}

func queryVariants(query string) []string {
	variants := []string{query}
	cleaned := strings.NewReplacer(
		"帮我看看", "",
		"帮忙看看", "",
		"帮我分析", "",
		"帮忙分析", "",
		"搜索", "",
		"查询", "",
		"查一下", "",
		"看看", "",
		"最近", "",
		"走势", "",
		"表现", "",
		"这个", "",
		"这只", "",
		"某个", "",
		"基金", "",
		"的", "",
		"？", "",
		"?", "",
		"，", "",
		",", "",
	).Replace(query)
	cleaned = strings.TrimSpace(cleaned)
	if cleaned != "" && cleaned != query {
		variants = append(variants, cleaned)
	}
	return uniqueStrings(variants)
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
