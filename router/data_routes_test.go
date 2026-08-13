package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupDashboardDataRouteTest(t *testing.T) (*gin.Engine, string) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Log{}, &model.Channel{}))

	originalDB := model.DB
	originalLogDB := model.LOG_DB
	originalMainType := common.MainDatabaseType()
	originalLogType := common.LogDatabaseType()
	originalRedisEnabled := common.RedisEnabled
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	originalGlobalRateLimitEnabled := common.GlobalApiRateLimitEnable
	model.DB = db
	model.LOG_DB = db
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	common.RedisEnabled = false
	common.MemoryCacheEnabled = false
	common.GlobalApiRateLimitEnable = false
	t.Cleanup(func() {
		model.DB = originalDB
		model.LOG_DB = originalLogDB
		common.SetDatabaseTypes(originalMainType, originalLogType)
		common.RedisEnabled = originalRedisEnabled
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
		common.GlobalApiRateLimitEnable = originalGlobalRateLimitEnabled
		sqlDB, closeErr := db.DB()
		if closeErr == nil {
			require.NoError(t, sqlDB.Close())
		}
	})

	accessToken := "ordinary-dashboard-token-0000000"
	require.Len(t, accessToken, 32)
	require.NoError(t, db.Create(&model.User{
		Id:          901,
		Username:    "route-user",
		Password:    "not-used-in-this-test",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		AccessToken: &accessToken,
		AuthVersion: 1,
	}).Error)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	SetApiRouter(engine)
	return engine, accessToken
}

func TestDashboardDataSelfRoutesAllowCommonUsersButAdminRoutesDoNot(t *testing.T) {
	engine, accessToken := setupDashboardDataRouteTest(t)

	selfRequest := httptest.NewRequest(http.MethodGet, "/api/data/channel-stats/self?start_timestamp=1000&end_timestamp=2000", nil)
	selfRequest.Header.Set("Authorization", "Bearer "+accessToken)
	selfResponse := httptest.NewRecorder()
	engine.ServeHTTP(selfResponse, selfRequest)
	assert.Equal(t, http.StatusOK, selfResponse.Code)

	adminRequest := httptest.NewRequest(http.MethodGet, "/api/data/channel-stats?start_timestamp=1000&end_timestamp=2000", nil)
	adminRequest.Header.Set("Authorization", "Bearer "+accessToken)
	adminResponse := httptest.NewRecorder()
	engine.ServeHTTP(adminResponse, adminRequest)
	assert.Equal(t, http.StatusForbidden, adminResponse.Code)
}

func TestDashboardDataRoutesMatchFrontendPaths(t *testing.T) {
	engine, _ := setupDashboardDataRouteTest(t)
	routes := make(map[string]struct{}, len(engine.Routes()))
	for _, route := range engine.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}

	expectedPaths := []string{
		"/api/data",
		"/api/data/daily-tokens",
		"/api/data/daily-tokens/self",
		"/api/data/daily-model-tokens",
		"/api/data/daily-model-tokens/self",
		"/api/data/channel-stats",
		"/api/data/channel-stats/self",
		"/api/data/subscription-usage",
		"/api/data/subscription-usage/self",
		"/api/data/subscription-model-usage",
		"/api/data/subscription-model-usage/self",
	}
	for _, path := range expectedPaths {
		_, ok := routes[http.MethodGet+" "+path]
		assert.True(t, ok, "missing GET %s", path)
	}
	_, hasTrailingSlashRoot := routes[http.MethodGet+" /api/data/"]
	assert.False(t, hasTrailingSlashRoot)
}
