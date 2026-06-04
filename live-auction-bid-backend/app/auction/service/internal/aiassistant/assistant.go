package aiassistant

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	v1 "live-auction-bid/backend/api/auction/service/v1"
)

const (
	ActionRevealTrustCard = "reveal_trust_card"
	ActionStartDuel       = "start_duel"
	ActionNavigate        = "navigate"
	ActionCopyText        = "copy_text"
)

type Config struct {
	Provider      string
	BaseURL       string
	Model         string
	APIKey        string
	Timeout       time.Duration
	MockOnFailure bool
}

type Assistant struct {
	cfg    Config
	client *http.Client
}

type BuyerConsultRequest struct {
	Query          string `json:"query"`
	RoomID         string `json:"roomId,omitempty"`
	LotID          string `json:"lotId,omitempty"`
	Budget         int64  `json:"budget,omitempty"`
	RiskPreference string `json:"riskPreference,omitempty"`
}

type BuyerConsultReply struct {
	Result       *v1.ReplyResult `json:"result,omitempty"`
	Answer       string          `json:"answer"`
	Intent       string          `json:"intent"`
	Results      []BuyerResult   `json:"results"`
	BidAdvice    BidAdvice       `json:"bidAdvice"`
	Sources      []Source        `json:"sources"`
	FallbackUsed bool            `json:"fallbackUsed"`
}

type BuyerResult struct {
	Type         string    `json:"type"`
	Title        string    `json:"title"`
	RoomID       string    `json:"roomId"`
	LotID        string    `json:"lotId"`
	Status       string    `json:"status"`
	CurrentPrice *v1.Money `json:"currentPrice,omitempty"`
	Href         string    `json:"href"`
	Reason       string    `json:"reason"`
}

type BidAdvice struct {
	NextBidAmount      *v1.Money `json:"nextBidAmount,omitempty"`
	MaxSuggestedAmount *v1.Money `json:"maxSuggestedAmount,omitempty"`
	Strategy           string    `json:"strategy"`
	Risks              []string  `json:"risks"`
	Confidence         float64   `json:"confidence"`
}

type Source struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	RoomID string `json:"roomId,omitempty"`
	LotID  string `json:"lotId,omitempty"`
}

type BuyerConsultContext struct {
	Candidates []LotCandidate   `json:"candidates"`
	Snapshot   *v1.RoomSnapshot `json:"snapshot,omitempty"`
}

type LotCandidate struct {
	Type         string    `json:"type"`
	Title        string    `json:"title"`
	RoomID       string    `json:"roomId"`
	LotID        string    `json:"lotId"`
	Status       string    `json:"status"`
	CurrentPrice *v1.Money `json:"currentPrice,omitempty"`
	Href         string    `json:"href"`
	Reason       string    `json:"reason"`
	Score        int       `json:"score,omitempty"`
	Lot          *v1.Lot   `json:"-"`
}

type MerchantAssistRequest struct {
	Page     string         `json:"page"`
	RoomID   string         `json:"roomId,omitempty"`
	LotID    string         `json:"lotId,omitempty"`
	Draft    map[string]any `json:"draft,omitempty"`
	Question string         `json:"question,omitempty"`
}

type MerchantAssistReply struct {
	Result             *v1.ReplyResult     `json:"result,omitempty"`
	Answer             string              `json:"answer"`
	Situation          *MerchantSituation  `json:"situation,omitempty"`
	TalkTracks         []string            `json:"talkTracks,omitempty"`
	Evidence           []string            `json:"evidence,omitempty"`
	Checklist          []ChecklistItem     `json:"checklist"`
	NextSteps          []string            `json:"nextSteps"`
	RecommendedActions []RecommendedAction `json:"recommendedActions"`
	DraftSuggestions   DraftSuggestions    `json:"draftSuggestions"`
	Warnings           []string            `json:"warnings"`
	FallbackUsed       bool                `json:"fallbackUsed"`
}

type MerchantSituation struct {
	Summary string            `json:"summary"`
	Metrics []SituationMetric `json:"metrics"`
}

type SituationMetric struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Tone  string `json:"tone,omitempty"`
}

type ChecklistItem struct {
	Label  string `json:"label"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type RecommendedAction struct {
	Type     string `json:"type"`
	Label    string `json:"label"`
	Reason   string `json:"reason"`
	Enabled  bool   `json:"enabled"`
	TargetID string `json:"targetId,omitempty"`
}

type DraftSuggestions struct {
	TitleSuggestion       string                `json:"titleSuggestion,omitempty"`
	DescriptionSuggestion string                `json:"descriptionSuggestion,omitempty"`
	Tags                  []string              `json:"tags,omitempty"`
	AfterSaleNote         string                `json:"afterSaleNote,omitempty"`
	TrustCards            []TrustCardSuggestion `json:"trustCards,omitempty"`
}

type TrustCardSuggestion struct {
	Type    string `json:"type"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type MerchantAssistContext struct {
	RoomID      string           `json:"roomId,omitempty"`
	CurrentLot  *v1.Lot          `json:"currentLot,omitempty"`
	Snapshot    *v1.RoomSnapshot `json:"snapshot,omitempty"`
	RankingSize int              `json:"rankingSize,omitempty"`
}

func New(cfg Config) *Assistant {
	cfg.Provider = strings.ToLower(strings.TrimSpace(cfg.Provider))
	if cfg.Provider == "" {
		cfg.Provider = "mock"
	}
	cfg.BaseURL = strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	cfg.Model = strings.TrimSpace(cfg.Model)
	cfg.APIKey = strings.TrimSpace(cfg.APIKey)
	if cfg.Timeout <= 0 {
		cfg.Timeout = 8 * time.Second
	}
	return &Assistant{cfg: cfg, client: &http.Client{Timeout: cfg.Timeout}}
}

func NewFromEnv(getenv func(string) string) *Assistant {
	timeout := 8 * time.Second
	if raw := strings.TrimSpace(getenv("AUCTION_AI_TIMEOUT")); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil {
			timeout = parsed
		}
	}
	assistant := New(Config{
		Provider:      getenv("AUCTION_AI_PROVIDER"),
		BaseURL:       getenv("AUCTION_AI_BASE_URL"),
		Model:         getenv("AUCTION_AI_MODEL"),
		APIKey:        getenv("AUCTION_AI_API_KEY"),
		Timeout:       timeout,
		MockOnFailure: parseBool(getenv("AUCTION_AI_MOCK_ON_FAILURE"), true),
	})
	slog.Info("ai assistant configured",
		"provider", assistant.cfg.Provider,
		"model", assistant.cfg.Model,
		"base_url_configured", assistant.cfg.BaseURL != "",
		"api_key_configured", assistant.cfg.APIKey != "",
		"timeout", assistant.cfg.Timeout.String(),
		"mock_on_failure", assistant.cfg.MockOnFailure,
	)
	return assistant
}

func (a *Assistant) ConsultBuyer(ctx context.Context, req BuyerConsultRequest, data BuyerConsultContext) (BuyerConsultReply, error) {
	fallback := fallbackBuyer(req, data)
	if a == nil || !a.configured() {
		return fallback, nil
	}
	var rawReply map[string]any
	err := a.chatJSON(ctx, "你是直播竞拍平台的买家侧 AI 竞拍咨询助手。必须只基于提供的候选拍品和来源回答，不编造场次、价格、链接。只输出 JSON。", map[string]any{
		"task":             "buyer_consult",
		"request":          req,
		"dynamic_context":  data,
		"static_knowledge": buyerKnowledge(),
		"schema":           "answer,intent,results,bidAdvice,sources,fallbackUsed",
	}, &rawReply)
	if err != nil {
		a.logFallback("buyer_consult", err)
		return fallback, nil
	}
	reply := normalizeBuyerReply(buyerReplyFromMap(rawReply), fallback)
	return reply, nil
}

func (a *Assistant) AssistMerchant(ctx context.Context, req MerchantAssistRequest, data MerchantAssistContext) (MerchantAssistReply, error) {
	fallback := fallbackMerchant(req, data)
	if a == nil || !a.configured() {
		return fallback, nil
	}
	var rawReply map[string]any
	err := a.chatJSON(ctx, "你是直播竞拍平台的商家端 AI 助手。你只能给建议，不能要求自动出价、自动落锤、自动取消或修改交易状态。只输出 JSON。", map[string]any{
		"task":             "merchant_assistant",
		"request":          req,
		"dynamic_context":  data,
		"static_knowledge": merchantKnowledge(),
		"safe_actions":     []string{ActionRevealTrustCard, ActionStartDuel, ActionNavigate, ActionCopyText},
		"schema":           "answer,situation,talkTracks,evidence,checklist,nextSteps,recommendedActions,draftSuggestions,warnings,fallbackUsed",
	}, &rawReply)
	if err != nil {
		a.logFallback("merchant_assistant", err)
		return fallback, nil
	}
	reply := normalizeMerchantReply(merchantReplyFromMap(rawReply), fallback)
	return reply, nil
}

func (a *Assistant) configured() bool {
	if a == nil {
		return false
	}
	return a.cfg.Provider == "deepseek" && a.cfg.BaseURL != "" && a.cfg.Model != "" && a.cfg.APIKey != ""
}

func (a *Assistant) chatJSON(ctx context.Context, system string, payload any, out any) error {
	userPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	reqBody := map[string]any{
		"model":           a.cfg.Model,
		"response_format": map[string]string{"type": "json_object"},
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": string(userPayload) + "\n请返回严格 JSON，不要 Markdown。"},
		},
		"temperature": 0.2,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	callCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), a.cfg.Timeout)
	defer cancel()

	endpoint := a.chatEndpoint()
	httpReq, err := http.NewRequestWithContext(callCtx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+a.cfg.APIKey)
	resp, err := a.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ai provider http status %d", resp.StatusCode)
	}
	var envelope struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return err
	}
	if len(envelope.Choices) == 0 || strings.TrimSpace(envelope.Choices[0].Message.Content) == "" {
		return errors.New("ai provider returned empty content")
	}
	content := extractJSONObject(envelope.Choices[0].Message.Content)
	if content == "" {
		return errors.New("ai provider returned non-json content")
	}
	return json.Unmarshal([]byte(content), out)
}

func (a *Assistant) chatEndpoint() string {
	base := strings.TrimRight(a.cfg.BaseURL, "/")
	if strings.HasSuffix(base, "/chat/completions") {
		return base
	}
	return base + "/chat/completions"
}

func (a *Assistant) logFallback(task string, err error) {
	if a == nil || err == nil {
		return
	}
	slog.Warn("ai assistant fallback",
		"task", task,
		"provider", a.cfg.Provider,
		"model", a.cfg.Model,
		"error_kind", classifyError(err),
	)
}

func classifyError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "deadline") || strings.Contains(err.Error(), "timeout") {
		return "timeout"
	}
	if strings.Contains(err.Error(), "json") {
		return "json"
	}
	if strings.Contains(err.Error(), "http status") {
		return "provider_http"
	}
	return "provider"
}

func extractJSONObject(content string) string {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	}
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start < 0 || end < start {
		return ""
	}
	return content[start : end+1]
}

func fallbackBuyer(req BuyerConsultRequest, data BuyerConsultContext) BuyerConsultReply {
	candidates := append([]LotCandidate(nil), data.Candidates...)
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].Score > candidates[j].Score })
	if len(candidates) > 5 {
		candidates = candidates[:5]
	}
	results := make([]BuyerResult, 0, len(candidates))
	sources := make([]Source, 0, len(candidates)+1)
	for _, candidate := range candidates {
		results = append(results, BuyerResult{
			Type:         firstNonEmpty(candidate.Type, "lot"),
			Title:        candidate.Title,
			RoomID:       candidate.RoomID,
			LotID:        candidate.LotID,
			Status:       candidate.Status,
			CurrentPrice: cloneMoney(candidate.CurrentPrice),
			Href:         candidate.Href,
			Reason:       candidate.Reason,
		})
		sources = append(sources, Source{Type: "lot", Title: candidate.Title, RoomID: candidate.RoomID, LotID: candidate.LotID})
	}
	advice := BidAdvice{Strategy: "先确认拍品证书、瑕疵和售后，再按最低加价参与；不要超过自己的预算上限。", Risks: []string{"非标品价格波动较大，请结合证书和瑕疵说明判断。"}, Confidence: 0.68}
	if len(candidates) > 0 && candidates[0].Lot != nil {
		advice = adviceForLot(candidates[0].Lot, req.Budget)
	}
	answer := "我会优先帮你找正在竞拍或即将开拍的公开拍品。"
	if len(results) == 0 {
		answer = "暂时没有找到匹配的公开竞拍场次，可以换个关键词或稍后再看。"
	}
	return BuyerConsultReply{
		Answer:       answer,
		Intent:       "find_auction",
		Results:      results,
		BidAdvice:    advice,
		Sources:      sources,
		FallbackUsed: true,
	}
}

func fallbackMerchant(req MerchantAssistRequest, data MerchantAssistContext) MerchantAssistReply {
	page := strings.ToLower(strings.TrimSpace(req.Page))
	if page == "" {
		page = "admin"
	}
	reply := MerchantAssistReply{
		Answer:       "AI 会按当前页面给出下一步建议，但所有竞拍动作仍需要你手动确认。",
		FallbackUsed: true,
	}
	switch page {
	case "create", "auction-create":
		reply.Answer = "先补齐拍品资料、图片、竞拍规则和信任讲解卡，再加入队列。"
		reply.Checklist = createChecklist(req.Draft)
		reply.NextSteps = []string{"上传主图和细节图", "填写标题、描述、分类和标签", "确认起拍价、加价幅度、时长和延时规则", "补齐证书、瑕疵、细节、售后四类讲解卡"}
		reply.DraftSuggestions = DraftSuggestions{
			TitleSuggestion:       "直播严选高价值竞拍拍品",
			DescriptionSuggestion: "适合直播竞拍展示的非标品，建议结合证书、细节图、瑕疵说明和售后承诺讲解。",
			Tags:                  []string{"直播竞拍", "严选", "信任卡"},
			AfterSaleNote:         "成交后按平台规则完成订单与模拟支付，售后以商家页面承诺为准。",
			TrustCards: []TrustCardSuggestion{
				{Type: "TRUST_CARD_TYPE_CERTIFICATE", Title: "证书信息", Content: "展示证书编号、鉴定机构和关键结论。"},
				{Type: "TRUST_CARD_TYPE_FLAW", Title: "瑕疵说明", Content: "如实说明磨损、划痕、缺件或使用痕迹。"},
				{Type: "TRUST_CARD_TYPE_DETAIL", Title: "细节展示", Content: "补充材质、尺寸、工艺和上手效果。"},
				{Type: "TRUST_CARD_TYPE_SERVICE", Title: "售后承诺", Content: "说明发货、支付、客服和退换边界。"},
			},
		}
	default:
		reply.Situation = controlSituation(data.CurrentLot, data.Snapshot)
		reply.TalkTracks = controlTalkTracks(data.CurrentLot, data.Snapshot)
		reply.Evidence = controlEvidence(data.CurrentLot, data.Snapshot)
		reply.Checklist = []ChecklistItem{{Label: "交易主链路", Status: "safe", Reason: "AI 只给建议，不改变竞拍状态。"}}
		reply.NextSteps = []string{"先同步房间快照", "观察排行榜和最近出价", "必要时手动展示信任卡或进入 Duel"}
		reply.RecommendedActions = controlActions(data.CurrentLot, data.RankingSize)
		reply.Warnings = controlWarnings(data.CurrentLot)
	}
	return reply
}

func normalizeBuyerReply(reply BuyerConsultReply, fallback BuyerConsultReply) BuyerConsultReply {
	if strings.TrimSpace(reply.Answer) == "" {
		reply.Answer = fallback.Answer
	}
	if strings.TrimSpace(reply.Intent) == "" {
		reply.Intent = fallback.Intent
	}
	if len(reply.Results) == 0 {
		reply.Results = fallback.Results
	}
	if reply.BidAdvice.Strategy == "" {
		reply.BidAdvice = fallback.BidAdvice
	}
	if len(reply.Sources) == 0 {
		reply.Sources = fallback.Sources
	}
	reply.FallbackUsed = false
	return reply
}

func buyerReplyFromMap(raw map[string]any) BuyerConsultReply {
	return BuyerConsultReply{
		Answer:    textFromMap(raw, "answer"),
		Intent:    textFromMap(raw, "intent"),
		Results:   buyerResultsFromAny(valueFromMap(raw, "results", "lots", "matches")),
		BidAdvice: bidAdviceFromAny(valueFromMap(raw, "bidAdvice", "bid_advice", "advice")),
		Sources:   sourcesFromAny(valueFromMap(raw, "sources", "source")),
	}
}

func buyerResultsFromAny(value any) []BuyerResult {
	items := sliceFromAny(value)
	out := make([]BuyerResult, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		title := textFromMap(m, "title", "name")
		roomID := textFromMap(m, "roomId", "room_id")
		lotID := textFromMap(m, "lotId", "lot_id")
		if title == "" && lotID == "" {
			continue
		}
		out = append(out, BuyerResult{
			Type:         firstNonEmpty(textFromMap(m, "type"), "lot"),
			Title:        title,
			RoomID:       roomID,
			LotID:        lotID,
			Status:       textFromMap(m, "status"),
			CurrentPrice: moneyFromAny(valueFromMap(m, "currentPrice", "current_price", "price")),
			Href:         textFromMap(m, "href", "url", "link"),
			Reason:       textFromMap(m, "reason", "description"),
		})
	}
	return out
}

func bidAdviceFromAny(value any) BidAdvice {
	m, ok := value.(map[string]any)
	if !ok {
		if strategy := textFromAny(value); strategy != "" {
			return BidAdvice{Strategy: strategy, Confidence: 0.5}
		}
		return BidAdvice{}
	}
	return BidAdvice{
		NextBidAmount:      moneyFromAny(valueFromMap(m, "nextBidAmount", "next_bid_amount", "nextBid", "next_bid")),
		MaxSuggestedAmount: moneyFromAny(valueFromMap(m, "maxSuggestedAmount", "max_suggested_amount", "maxAmount", "max_amount")),
		Strategy:           textFromMap(m, "strategy", "advice", "summary"),
		Risks:              stringSliceFromAny(valueFromMap(m, "risks", "risk")),
		Confidence:         floatFromAny(valueFromMap(m, "confidence"), 0),
	}
}

func sourcesFromAny(value any) []Source {
	items := sliceFromAny(value)
	if len(items) == 0 {
		if title := textFromAny(value); title != "" {
			return []Source{{Type: "context", Title: title}}
		}
		return nil
	}
	out := make([]Source, 0, len(items))
	for _, item := range items {
		if title := textFromAny(item); title != "" {
			out = append(out, Source{Type: "context", Title: title})
			continue
		}
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		title := textFromMap(m, "title", "name")
		if title == "" {
			continue
		}
		out = append(out, Source{
			Type:   firstNonEmpty(textFromMap(m, "type"), "context"),
			Title:  title,
			RoomID: textFromMap(m, "roomId", "room_id"),
			LotID:  textFromMap(m, "lotId", "lot_id"),
		})
	}
	return out
}

func normalizeMerchantReply(reply MerchantAssistReply, fallback MerchantAssistReply) MerchantAssistReply {
	if strings.TrimSpace(reply.Answer) == "" {
		reply.Answer = fallback.Answer
	}
	if reply.Situation == nil || strings.TrimSpace(reply.Situation.Summary) == "" {
		reply.Situation = fallback.Situation
	}
	if len(reply.TalkTracks) == 0 {
		reply.TalkTracks = fallback.TalkTracks
	}
	if len(reply.Evidence) == 0 {
		reply.Evidence = fallback.Evidence
	}
	if len(reply.Checklist) == 0 {
		reply.Checklist = fallback.Checklist
	}
	if len(reply.NextSteps) == 0 {
		reply.NextSteps = fallback.NextSteps
	}
	reply.RecommendedActions = sanitizeActions(reply.RecommendedActions)
	if len(reply.RecommendedActions) == 0 {
		reply.RecommendedActions = fallback.RecommendedActions
	}
	if len(reply.DraftSuggestions.TrustCards) == 0 && fallback.DraftSuggestions.TitleSuggestion != "" {
		reply.DraftSuggestions = fallback.DraftSuggestions
	}
	if len(reply.Warnings) == 0 {
		reply.Warnings = fallback.Warnings
	}
	reply.FallbackUsed = false
	return reply
}

func merchantReplyFromMap(raw map[string]any) MerchantAssistReply {
	return MerchantAssistReply{
		Answer:             textFromMap(raw, "answer"),
		Situation:          situationFromAny(valueFromMap(raw, "situation", "scene", "status")),
		TalkTracks:         stringSliceFromAny(valueFromMap(raw, "talkTracks", "talk_tracks", "scripts")),
		Evidence:           stringSliceFromAny(valueFromMap(raw, "evidence", "sources", "basis")),
		Checklist:          checklistFromAny(valueFromMap(raw, "checklist")),
		NextSteps:          stringSliceFromAny(valueFromMap(raw, "nextSteps", "next_steps", "nextStep")),
		RecommendedActions: actionsFromAny(valueFromMap(raw, "recommendedActions", "recommended_actions", "actions")),
		DraftSuggestions:   draftSuggestionsFromAny(valueFromMap(raw, "draftSuggestions", "draft_suggestions", "suggestions")),
		Warnings:           stringSliceFromAny(valueFromMap(raw, "warnings", "warning", "risks")),
	}
}

func situationFromAny(value any) *MerchantSituation {
	if text := textFromAny(value); text != "" {
		return &MerchantSituation{Summary: text}
	}
	m, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	situation := &MerchantSituation{
		Summary: firstNonEmpty(textFromMap(m, "summary"), textFromMap(m, "answer"), textFromMap(m, "title")),
		Metrics: situationMetricsFromAny(valueFromMap(m, "metrics", "items", "facts")),
	}
	if situation.Summary == "" && len(situation.Metrics) == 0 {
		return nil
	}
	return situation
}

func situationMetricsFromAny(value any) []SituationMetric {
	items := sliceFromAny(value)
	out := make([]SituationMetric, 0, len(items))
	for _, item := range items {
		if text := textFromAny(item); text != "" {
			out = append(out, SituationMetric{Label: "提示", Value: text})
			continue
		}
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		label := firstNonEmpty(textFromMap(m, "label"), textFromMap(m, "name"), textFromMap(m, "title"))
		value := firstNonEmpty(textFromMap(m, "value"), textFromMap(m, "text"), textFromMap(m, "summary"))
		if label == "" || value == "" {
			continue
		}
		out = append(out, SituationMetric{Label: label, Value: value, Tone: textFromMap(m, "tone")})
	}
	return out
}

func checklistFromAny(value any) []ChecklistItem {
	items := sliceFromAny(value)
	if len(items) == 0 {
		if label := textFromAny(value); label != "" {
			return []ChecklistItem{{Label: label, Status: "todo", Reason: "模型建议"}}
		}
		return nil
	}
	out := make([]ChecklistItem, 0, len(items))
	for _, item := range items {
		if label := textFromAny(item); label != "" {
			out = append(out, ChecklistItem{Label: label, Status: "todo", Reason: "模型建议"})
			continue
		}
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		label := firstNonEmpty(textFromMap(m, "label"), textFromMap(m, "title"), textFromMap(m, "text"), textFromMap(m, "name"))
		if label == "" {
			continue
		}
		out = append(out, ChecklistItem{
			Label:  label,
			Status: firstNonEmpty(textFromMap(m, "status"), "todo"),
			Reason: firstNonEmpty(textFromMap(m, "reason"), textFromMap(m, "description"), "模型建议"),
		})
	}
	return out
}

func actionsFromAny(value any) []RecommendedAction {
	items := sliceFromAny(value)
	out := make([]RecommendedAction, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		actionType := textFromMap(m, "type", "actionType", "action")
		if actionType == "" {
			continue
		}
		out = append(out, RecommendedAction{
			Type:     actionType,
			Label:    firstNonEmpty(textFromMap(m, "label"), textFromMap(m, "title"), actionType),
			Reason:   textFromMap(m, "reason", "description"),
			Enabled:  boolFromAny(valueFromMap(m, "enabled"), true),
			TargetID: textFromMap(m, "targetId", "target_id", "cardId", "lotId"),
		})
	}
	return out
}

func draftSuggestionsFromAny(value any) DraftSuggestions {
	m, ok := value.(map[string]any)
	if !ok {
		return DraftSuggestions{}
	}
	return DraftSuggestions{
		TitleSuggestion:       textFromMap(m, "titleSuggestion", "title_suggestion", "title"),
		DescriptionSuggestion: textFromMap(m, "descriptionSuggestion", "description_suggestion", "description"),
		Tags:                  stringSliceFromAny(valueFromMap(m, "tags")),
		AfterSaleNote:         textFromMap(m, "afterSaleNote", "after_sale_note", "afterSaleNotes", "after_sale_notes", "afterSale"),
		TrustCards:            trustCardSuggestionsFromAny(valueFromMap(m, "trustCards", "trust_cards")),
	}
}

func trustCardSuggestionsFromAny(value any) []TrustCardSuggestion {
	items := sliceFromAny(value)
	out := make([]TrustCardSuggestion, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		title := textFromMap(m, "title", "label")
		content := textFromMap(m, "content", "text", "description")
		if title == "" && content == "" {
			continue
		}
		out = append(out, TrustCardSuggestion{
			Type:    textFromMap(m, "type", "cardType", "card_type"),
			Title:   title,
			Content: content,
		})
	}
	return out
}

func sanitizeActions(actions []RecommendedAction) []RecommendedAction {
	out := make([]RecommendedAction, 0, len(actions))
	for _, action := range actions {
		switch action.Type {
		case ActionRevealTrustCard, ActionStartDuel, ActionNavigate, ActionCopyText:
			out = append(out, action)
		}
	}
	return out
}

func valueFromMap(m map[string]any, keys ...string) any {
	if m == nil {
		return nil
	}
	for _, key := range keys {
		if value, ok := m[key]; ok {
			return value
		}
	}
	return nil
}

func textFromMap(m map[string]any, keys ...string) string {
	return textFromAny(valueFromMap(m, keys...))
}

func textFromAny(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case float64, bool:
		return strings.TrimSpace(fmt.Sprint(v))
	default:
		return ""
	}
}

func sliceFromAny(value any) []any {
	switch v := value.(type) {
	case nil:
		return nil
	case []any:
		return v
	case []string:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, item)
		}
		return out
	default:
		return nil
	}
}

func stringSliceFromAny(value any) []string {
	if text := textFromAny(value); text != "" {
		return []string{text}
	}
	items := sliceFromAny(value)
	out := make([]string, 0, len(items))
	for _, item := range items {
		if text := textFromAny(item); text != "" {
			out = append(out, text)
		}
	}
	return out
}

func boolFromAny(value any, fallback bool) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return parseBool(v, fallback)
	default:
		return fallback
	}
}

func floatFromAny(value any, fallback float64) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return fallback
	}
}

func int64FromAny(value any) int64 {
	switch v := value.(type) {
	case float64:
		return int64(v)
	case int:
		return int64(v)
	case int64:
		return v
	default:
		return 0
	}
}

func moneyFromAny(value any) *v1.Money {
	switch v := value.(type) {
	case nil:
		return nil
	case map[string]any:
		amount := int64FromAny(valueFromMap(v, "amount", "value"))
		if amount <= 0 {
			return nil
		}
		return &v1.Money{Amount: amount, Currency: firstNonEmpty(textFromMap(v, "currency"), "CNY")}
	case float64, int, int64:
		amount := int64FromAny(v)
		if amount <= 0 {
			return nil
		}
		return &v1.Money{Amount: amount, Currency: "CNY"}
	default:
		return nil
	}
}

func adviceForLot(lot *v1.Lot, budget int64) BidAdvice {
	current := moneyAmount(lot.GetCurrentPrice())
	if current <= 0 {
		current = moneyAmount(lot.GetRule().GetStartPrice())
	}
	increment := moneyAmount(lot.GetRule().GetMinIncrement())
	if increment <= 0 {
		increment = 100
	}
	currency := firstNonEmpty(lot.GetCurrentPrice().GetCurrency(), lot.GetRule().GetStartPrice().GetCurrency(), "CNY")
	next := current + increment
	maxSuggested := next + increment*2
	if budget > 0 && budget < maxSuggested {
		maxSuggested = budget
	}
	if cap := moneyAmount(lot.GetRule().GetCapPrice()); cap > 0 && cap < maxSuggested {
		maxSuggested = cap
	}
	return BidAdvice{
		NextBidAmount:      &v1.Money{Amount: next, Currency: currency},
		MaxSuggestedAmount: &v1.Money{Amount: maxSuggested, Currency: currency},
		Strategy:           "建议从下一口价开始，观察 Top2 价差和剩余时间；超过心理预算就停止。",
		Risks:              []string{"竞拍价可能快速上升", "请先查看证书、瑕疵和售后信任卡"},
		Confidence:         0.72,
	}
}

func createChecklist(draft map[string]any) []ChecklistItem {
	return []ChecklistItem{
		{Label: "拍品标题", Status: draftStatus(draft, "title"), Reason: "标题会影响搜索和直播间识别。"},
		{Label: "拍品图片", Status: draftStatus(draft, "imageUrl"), Reason: "主图是用户进入竞拍的第一判断。"},
		{Label: "竞拍规则", Status: draftStatus(draft, "rule"), Reason: "起拍价、加价幅度、时长会影响出价校验。"},
		{Label: "信任讲解卡", Status: draftStatus(draft, "trustCards"), Reason: "证书、瑕疵、细节、售后用于建立非标品信任。"},
	}
}

func draftStatus(draft map[string]any, key string) string {
	if draft == nil {
		return "missing"
	}
	value, ok := draft[key]
	if !ok || value == nil || strings.TrimSpace(fmt.Sprint(value)) == "" {
		return "missing"
	}
	return "ready"
}

func controlSituation(lot *v1.Lot, snapshot *v1.RoomSnapshot) *MerchantSituation {
	if lot == nil {
		return &MerchantSituation{
			Summary: "当前没有直播中的拍品，AI 控场助手只提示准备动作。",
			Metrics: []SituationMetric{
				{Label: "当前竞拍", Value: "无 LIVE", Tone: "warning"},
				{Label: "建议", Value: "回到本场队列开拍", Tone: "info"},
			},
		}
	}
	ranking := snapshot.GetRanking()
	recentBids := snapshot.GetRecentBids()
	unrevealed := unrevealedTrustCardCount(lot)
	topGap := "暂无 Top2"
	if len(ranking) >= 2 {
		gap := moneyAmount(ranking[0].GetAmount()) - moneyAmount(ranking[1].GetAmount())
		if gap < 0 {
			gap = 0
		}
		topGap = moneyText(&v1.Money{Amount: gap, Currency: firstNonEmpty(ranking[0].GetAmount().GetCurrency(), "CNY")})
	}
	leftText := "等待快照"
	if snapshot.GetServerTimeUnixMs() > 0 && lot.GetEndsAtUnixMs() > 0 {
		left := (lot.GetEndsAtUnixMs() - snapshot.GetServerTimeUnixMs()) / 1000
		if left < 0 {
			left = 0
		}
		leftText = fmt.Sprintf("%ds", left)
	}
	return &MerchantSituation{
		Summary: fmt.Sprintf("当前拍品「%s」正在竞拍，AI 建议优先围绕信任信息、Top2 竞争和剩余时间控场。", lot.GetTitle()),
		Metrics: []SituationMetric{
			{Label: "当前价", Value: moneyText(lot.GetCurrentPrice()), Tone: "success"},
			{Label: "参与/出价", Value: fmt.Sprintf("%d/%d", lot.GetStats().GetParticipantCount(), lot.GetStats().GetBidCount()), Tone: "info"},
			{Label: "Top2 价差", Value: topGap, Tone: "warning"},
			{Label: "最近出价", Value: fmt.Sprintf("%d 条", len(recentBids)), Tone: "info"},
			{Label: "未揭示信任卡", Value: fmt.Sprintf("%d 张", unrevealed), Tone: trustCardTone(unrevealed)},
			{Label: "剩余时间", Value: leftText, Tone: "warning"},
			{Label: "Duel", Value: duelText(lot), Tone: "purple"},
		},
	}
}

func controlTalkTracks(lot *v1.Lot, snapshot *v1.RoomSnapshot) []string {
	if lot == nil {
		return []string{"当前还没有直播中拍品，可以先回到本场队列确认下一件拍品的图片、规则和信任卡。"}
	}
	tracks := []string{
		fmt.Sprintf("现在这件「%s」当前价是 %s，想参与的朋友可以先看清楚加价幅度和预算上限。", lot.GetTitle(), moneyText(lot.GetCurrentPrice())),
	}
	if unrevealedTrustCardCount(lot) > 0 {
		tracks = append(tracks, "我先把证书、瑕疵或售后信息展示给大家，大家确认清楚再决定是否继续出价。")
	}
	ranking := snapshot.GetRanking()
	if len(ranking) >= 2 {
		tracks = append(tracks, fmt.Sprintf("现在前两名咬得比较紧，Top2 价差是 %s，最后阶段可以理性跟价。", top2GapText(ranking)))
	}
	if len(ranking) >= 2 && !lot.GetDuelState().GetActive() {
		tracks = append(tracks, "如果大家还想继续争，可以进入 Duel 模式，但所有出价仍按平台规则校验。")
	}
	return tracks
}

func controlEvidence(lot *v1.Lot, snapshot *v1.RoomSnapshot) []string {
	if lot == nil {
		return []string{"依据：当前 RoomSnapshot 未返回直播中拍品。"}
	}
	return []string{
		fmt.Sprintf("依据：当前拍品状态 %s，当前价 %s。", lot.GetStatus().String(), moneyText(lot.GetCurrentPrice())),
		fmt.Sprintf("依据：排行榜 %d 人，最近出价 %d 条。", len(snapshot.GetRanking()), len(snapshot.GetRecentBids())),
		fmt.Sprintf("依据：信任卡共 %d 张，未揭示 %d 张。", len(lot.GetTrustCards()), unrevealedTrustCardCount(lot)),
	}
}

func controlActions(lot *v1.Lot, rankingSize int) []RecommendedAction {
	if lot == nil {
		return []RecommendedAction{{Type: ActionNavigate, Label: "查看本场队列", Reason: "当前没有直播中拍品，先准备下一件拍品。", Enabled: true, TargetID: "/admin/auctions"}}
	}
	actions := make([]RecommendedAction, 0, 2)
	for _, card := range lot.GetTrustCards() {
		if card != nil && !card.GetRevealed() {
			actions = append(actions, RecommendedAction{Type: ActionRevealTrustCard, Label: "展示信任卡", Reason: "当前还有未揭示的证书/瑕疵/售后信息，可增强出价信心。", Enabled: true, TargetID: card.GetId()})
			break
		}
	}
	if rankingSize >= 2 && !lot.GetDuelState().GetActive() {
		actions = append(actions, RecommendedAction{Type: ActionStartDuel, Label: "进入 Duel", Reason: "Top2 已形成竞争，可以增强最后阶段竞拍氛围。", Enabled: true, TargetID: lot.GetId()})
	}
	return actions
}

func controlWarnings(lot *v1.Lot) []string {
	if lot == nil {
		return []string{"当前没有直播中拍品，AI 不会触发任何竞拍动作。"}
	}
	warnings := []string{"AI 建议只作为控场参考，落锤、取消和订单仍由原系统规则处理。"}
	if len(lot.GetTrustCards()) == 0 {
		warnings = append(warnings, "拍品缺少信任讲解卡，可能影响用户出价信心。")
	}
	return warnings
}

func unrevealedTrustCardCount(lot *v1.Lot) int {
	if lot == nil {
		return 0
	}
	count := 0
	for _, card := range lot.GetTrustCards() {
		if card != nil && !card.GetRevealed() {
			count += 1
		}
	}
	return count
}

func trustCardTone(count int) string {
	if count > 0 {
		return "warning"
	}
	return "success"
}

func duelText(lot *v1.Lot) string {
	if lot.GetDuelState().GetActive() {
		return "进行中"
	}
	return "可按局势手动开启"
}

func top2GapText(ranking []*v1.RankingItem) string {
	if len(ranking) < 2 {
		return "暂无"
	}
	gap := moneyAmount(ranking[0].GetAmount()) - moneyAmount(ranking[1].GetAmount())
	if gap < 0 {
		gap = 0
	}
	return moneyText(&v1.Money{Amount: gap, Currency: firstNonEmpty(ranking[0].GetAmount().GetCurrency(), "CNY")})
}

func buyerKnowledge() []string {
	return []string{
		"公开可见拍品只包含即将开始、直播中、延时中的拍品。",
		"用户出价必须不低于当前价加最低加价幅度。",
		"达到封顶价会自动成交，结束前出价可能触发反狙击延时。",
	}
}

func merchantKnowledge() []string {
	return []string{
		"商家创建拍品后需要加入队列，再由中控台开拍。",
		"信任讲解卡用于展示证书、瑕疵、细节和售后。",
		"Duel 只强化 Top2 竞拍氛围，不改变合法出价规则。",
		"AI 只能输出建议，不能写入 lot/order/payment 状态。",
	}
}

func cloneMoney(m *v1.Money) *v1.Money {
	if m == nil {
		return nil
	}
	return &v1.Money{Amount: m.GetAmount(), Currency: m.GetCurrency()}
}

func moneyAmount(m *v1.Money) int64 {
	if m == nil {
		return 0
	}
	return m.GetAmount()
}

func moneyText(m *v1.Money) string {
	if m == nil {
		return "暂无"
	}
	currency := firstNonEmpty(m.GetCurrency(), "CNY")
	if currency == "CNY" {
		return fmt.Sprintf("¥%.2f", float64(m.GetAmount())/100)
	}
	return fmt.Sprintf("%s %.2f", currency, float64(m.GetAmount())/100)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func parseBool(raw string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "":
		return fallback
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}
