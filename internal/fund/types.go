package fund

type SearchCandidate struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type Profile struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	Company      string `json:"company"`
	LatestNAV    string `json:"latestNav"`
	DailyChange  string `json:"dailyChange"`
	AsOfDate     string `json:"asOfDate"`
	EstimatedNAV string `json:"estimatedNav"`
}

type TrendPoint struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

type Trend struct {
	Range  string       `json:"range"`
	Points []TrendPoint `json:"points"`
}

type QueryResult struct {
	Mode                      string            `json:"mode"`
	Query                     string            `json:"query"`
	Candidates                []SearchCandidate `json:"candidates,omitempty"`
	Profile                   *Profile          `json:"profile,omitempty"`
	Trend                     *Trend            `json:"trend,omitempty"`
	RecommendationPlaceholder string            `json:"recommendationPlaceholder,omitempty"`
	AnalysisPlaceholder       string            `json:"analysisPlaceholder,omitempty"`
}
