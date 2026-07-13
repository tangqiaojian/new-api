package model

// SubscriptionDailyUsage represents daily token usage billed from subscriptions.
// Each row aggregates one day + one subscription/plan combination so the frontend
// can render a per-day, per-subscription breakdown of subscription-billed traffic.
type SubscriptionDailyUsage struct {
	Date             string `json:"date"`
	SubscriptionId   int    `json:"subscription_id"`
	PlanId           int    `json:"plan_id"`
	PlanTitle        string `json:"plan_title"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TotalTokens      int    `json:"total_tokens"`
	CachedTokens     int    `json:"cached_tokens"`
	RequestCount     int    `json:"request_count"`
	Quota            int    `json:"quota"`
}

// SubscriptionModelUsage represents per-model token usage billed from subscriptions.
// Each row aggregates one model + one plan combination so the frontend can render a
// per-model breakdown of subscription-billed traffic.
type SubscriptionModelUsage struct {
	ModelName        string `json:"model_name"`
	PlanId           int    `json:"plan_id"`
	PlanTitle        string `json:"plan_title"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TotalTokens      int    `json:"total_tokens"`
	CachedTokens     int    `json:"cached_tokens"`
	RequestCount     int    `json:"request_count"`
	Quota            int    `json:"quota"`
}

// subscriptionBillingSourceFilter returns the WHERE fragment that restricts logs to
// those billed from a subscription. The `other` JSON is written compactly (via
// json.Marshal, no spaces), so a LIKE substring match is reliable across all DBs.
func subscriptionBillingSourceFilter() string {
	return LogOtherCol() + ` LIKE '%"billing_source":"subscription"%'`
}

// subscriptionTotalTokensExpr builds the total_tokens expression. When includeCache is
// true, cached tokens (extracted from the `other` JSON) are added on top of
// prompt_tokens + completion_tokens.
func subscriptionTotalTokensExpr(includeCache bool) string {
	expr := "COALESCE(SUM(prompt_tokens), 0) + COALESCE(SUM(completion_tokens), 0)"
	if includeCache {
		expr += " + " + logCacheTokensSumExpr()
	}
	return expr
}

// subscriptionDailySelectColumns builds the SELECT clause for daily subscription
// usage aggregation. plan_title is wrapped in MAX() because it is a string extracted
// from JSON; it is functionally dependent on subscription_id/plan_id (which are in
// GROUP BY), so MAX() simply picks the single distinct value without requiring the
// string expression itself to appear in GROUP BY.
func subscriptionDailySelectColumns(dateExpr string, includeCache bool) string {
	return dateExpr + " as date, " +
		LogJsonExtractInt("subscription_id") + " as subscription_id, " +
		LogJsonExtractInt("subscription_plan_id") + " as plan_id, " +
		"MAX(" + LogJsonExtractString("subscription_plan_title") + ") as plan_title, " +
		"COALESCE(SUM(prompt_tokens), 0) as prompt_tokens, " +
		"COALESCE(SUM(completion_tokens), 0) as completion_tokens, " +
		subscriptionTotalTokensExpr(includeCache) + " as total_tokens, " +
		logCacheTokensSumExpr() + " as cached_tokens, " +
		"COUNT(*) as request_count, " +
		"COALESCE(SUM(quota), 0) as quota"
}

// subscriptionModelSelectColumns builds the SELECT clause for per-model subscription
// usage aggregation.
func subscriptionModelSelectColumns(includeCache bool) string {
	return "model_name, " +
		LogJsonExtractInt("subscription_plan_id") + " as plan_id, " +
		"MAX(" + LogJsonExtractString("subscription_plan_title") + ") as plan_title, " +
		"COALESCE(SUM(prompt_tokens), 0) as prompt_tokens, " +
		"COALESCE(SUM(completion_tokens), 0) as completion_tokens, " +
		subscriptionTotalTokensExpr(includeCache) + " as total_tokens, " +
		logCacheTokensSumExpr() + " as cached_tokens, " +
		"COUNT(*) as request_count, " +
		"COALESCE(SUM(quota), 0) as quota"
}

// GetSelfSubscriptionDailyUsage returns daily token usage billed from subscriptions
// for a specific user. Results are grouped by date + subscription_id + plan_id so each
// row represents one day's usage under one subscription/plan.
//
// Filters:
//   - startTime/endTime: created_at range (unix seconds, inclusive)
//   - subscriptionId: when > 0, restrict to that subscription (via JSON extract, which
//     avoids the false-positive problem of a LIKE on a numeric substring)
//   - model: when non-empty, restrict to that model_name
//
// When includeCache is true, total_tokens adds cached_tokens on top of
// prompt_tokens + completion_tokens. cached_tokens is always populated from the
// `other` JSON's cache_tokens field.
func GetSelfSubscriptionDailyUsage(userId int, startTime, endTime int64, subscriptionId int, modelName string, includeCache bool) ([]*SubscriptionDailyUsage, error) {
	var data []*SubscriptionDailyUsage
	dateExpr := dailyTokenDateExpression()

	query := LOG_DB.Table("logs").
		Select(subscriptionDailySelectColumns(dateExpr, includeCache)).
		Where("user_id = ? AND type = ? AND created_at >= ? AND created_at <= ?", userId, LogTypeConsume, startTime, endTime).
		Where(subscriptionBillingSourceFilter())

	if subscriptionId > 0 {
		// Use the JSON-extract comparison rather than a LIKE on the numeric value:
		// a LIKE such as '%"subscription_id":5%' would also match ...:50, ...:55, etc.
		query = query.Where(LogJsonExtractInt("subscription_id")+" = ?", subscriptionId)
	}
	if modelName != "" {
		query = query.Where("model_name = ?", modelName)
	}

	// Group by date + subscription_id + plan_id. When a specific subscriptionId is
	// filtered, subscription_id/plan_id are constant per row, so including them in
	// GROUP BY is harmless and keeps the query shape uniform across DBs. plan_title
	// is NOT in GROUP BY (it is selected via MAX), avoiding dialect-specific issues
	// with grouping by a string-typed JSON extract.
	groupBy := dateExpr + ", " +
		LogJsonExtractInt("subscription_id") + ", " +
		LogJsonExtractInt("subscription_plan_id")

	err := query.
		Group(groupBy).
		Order("date DESC, total_tokens DESC").
		Find(&data).Error
	if err != nil {
		return nil, err
	}
	return data, nil
}

// GetSelfSubscriptionModelUsage returns per-model token usage billed from
// subscriptions for a specific user. Results are grouped by model_name + plan_id so
// each row represents one model's usage under one plan.
//
// The filters and includeCache semantics mirror GetSelfSubscriptionDailyUsage.
func GetSelfSubscriptionModelUsage(userId int, startTime, endTime int64, subscriptionId int, modelName string, includeCache bool) ([]*SubscriptionModelUsage, error) {
	var data []*SubscriptionModelUsage

	query := LOG_DB.Table("logs").
		Select(subscriptionModelSelectColumns(includeCache)).
		Where("user_id = ? AND type = ? AND created_at >= ? AND created_at <= ?", userId, LogTypeConsume, startTime, endTime).
		Where(subscriptionBillingSourceFilter())

	if subscriptionId > 0 {
		query = query.Where(LogJsonExtractInt("subscription_id")+" = ?", subscriptionId)
	}
	if modelName != "" {
		query = query.Where("model_name = ?", modelName)
	}

	groupBy := "model_name, " + LogJsonExtractInt("subscription_plan_id")

	err := query.
		Group(groupBy).
		Order("total_tokens DESC").
		Find(&data).Error
	if err != nil {
		return nil, err
	}
	return data, nil
}
