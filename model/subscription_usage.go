package model

// SubscriptionDailyUsage contains daily usage billed from a subscription.
type SubscriptionDailyUsage struct {
	Date             string `json:"date"`
	SubscriptionID   int    `json:"subscription_id"`
	PlanID           int    `json:"plan_id"`
	PlanTitle        string `json:"plan_title"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TotalTokens      int    `json:"total_tokens"`
	CachedTokens     int    `json:"cached_tokens"`
	RequestCount     int    `json:"request_count"`
	Quota            int    `json:"quota"`
}

// SubscriptionModelUsage contains per-model usage billed from subscriptions.
type SubscriptionModelUsage struct {
	ModelName        string `json:"model_name"`
	PlanID           int    `json:"plan_id"`
	PlanTitle        string `json:"plan_title"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TotalTokens      int    `json:"total_tokens"`
	CachedTokens     int    `json:"cached_tokens"`
	RequestCount     int    `json:"request_count"`
	Quota            int    `json:"quota"`
}

func subscriptionTotalTokensExpression(includeCache bool) string {
	expression := "COALESCE(SUM(prompt_tokens), 0) + COALESCE(SUM(completion_tokens), 0)"
	if includeCache {
		expression += " + " + logCacheTokensSumExpr()
	}
	return expression
}

func subscriptionDailySelectColumns(dateExpression string, includeCache bool) string {
	return dateExpression + " AS date, " +
		logJSONIntExpression("subscription_id") + " AS subscription_id, " +
		logJSONIntExpression("subscription_plan_id") + " AS plan_id, " +
		"MAX(" + logJSONStringExpression("subscription_plan_title") + ") AS plan_title, " +
		"COALESCE(SUM(prompt_tokens), 0) AS prompt_tokens, " +
		"COALESCE(SUM(completion_tokens), 0) AS completion_tokens, " +
		subscriptionTotalTokensExpression(includeCache) + " AS total_tokens, " +
		logCacheTokensSumExpr() + " AS cached_tokens, " +
		"COUNT(*) AS request_count, " +
		"COALESCE(SUM(quota), 0) AS quota"
}

func subscriptionModelSelectColumns(includeCache bool) string {
	return "model_name, " +
		logJSONIntExpression("subscription_plan_id") + " AS plan_id, " +
		"MAX(" + logJSONStringExpression("subscription_plan_title") + ") AS plan_title, " +
		"COALESCE(SUM(prompt_tokens), 0) AS prompt_tokens, " +
		"COALESCE(SUM(completion_tokens), 0) AS completion_tokens, " +
		subscriptionTotalTokensExpression(includeCache) + " AS total_tokens, " +
		logCacheTokensSumExpr() + " AS cached_tokens, " +
		"COUNT(*) AS request_count, " +
		"COALESCE(SUM(quota), 0) AS quota"
}

// GetSubscriptionDailyUsage returns daily subscription-funded usage. A zero userID
// selects all users for the administrative view.
func GetSubscriptionDailyUsage(userID int, startTime int64, endTime int64, subscriptionID int, modelName string, includeCache bool) ([]*SubscriptionDailyUsage, error) {
	data := make([]*SubscriptionDailyUsage, 0)
	dateExpression := dailyTokenDateExpression()
	query := LOG_DB.Table("logs").
		Select(subscriptionDailySelectColumns(dateExpression, includeCache)).
		Where("type = ? AND created_at >= ? AND created_at <= ?", LogTypeConsume, startTime, endTime).
		Where(logJSONStringExpression("billing_source")+" = ?", "subscription")
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	if subscriptionID > 0 {
		query = query.Where(logJSONIntExpression("subscription_id")+" = ?", subscriptionID)
	}
	if modelName != "" {
		query = query.Where("model_name = ?", modelName)
	}

	groupExpression := dateExpression + ", " +
		logJSONIntExpression("subscription_id") + ", " +
		logJSONIntExpression("subscription_plan_id")
	err := query.
		Group(groupExpression).
		Order("date DESC, total_tokens DESC").
		Find(&data).Error
	return data, err
}

// GetSelfSubscriptionDailyUsage returns daily subscription-funded usage for one user.
func GetSelfSubscriptionDailyUsage(userID int, startTime int64, endTime int64, subscriptionID int, modelName string, includeCache bool) ([]*SubscriptionDailyUsage, error) {
	if userID <= 0 {
		return []*SubscriptionDailyUsage{}, nil
	}
	return GetSubscriptionDailyUsage(userID, startTime, endTime, subscriptionID, modelName, includeCache)
}

// GetSubscriptionModelUsage returns subscription-funded usage grouped by model and plan.
func GetSubscriptionModelUsage(userID int, startTime int64, endTime int64, subscriptionID int, modelName string, includeCache bool) ([]*SubscriptionModelUsage, error) {
	data := make([]*SubscriptionModelUsage, 0)
	query := LOG_DB.Table("logs").
		Select(subscriptionModelSelectColumns(includeCache)).
		Where("type = ? AND created_at >= ? AND created_at <= ?", LogTypeConsume, startTime, endTime).
		Where(logJSONStringExpression("billing_source")+" = ?", "subscription")
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	if subscriptionID > 0 {
		query = query.Where(logJSONIntExpression("subscription_id")+" = ?", subscriptionID)
	}
	if modelName != "" {
		query = query.Where("model_name = ?", modelName)
	}

	err := query.
		Group("model_name, " + logJSONIntExpression("subscription_plan_id")).
		Order("total_tokens DESC, model_name ASC").
		Find(&data).Error
	return data, err
}

// GetSelfSubscriptionModelUsage returns subscription-funded model usage for one user.
func GetSelfSubscriptionModelUsage(userID int, startTime int64, endTime int64, subscriptionID int, modelName string, includeCache bool) ([]*SubscriptionModelUsage, error) {
	if userID <= 0 {
		return []*SubscriptionModelUsage{}, nil
	}
	return GetSubscriptionModelUsage(userID, startTime, endTime, subscriptionID, modelName, includeCache)
}
