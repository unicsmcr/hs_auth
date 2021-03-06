package routers

import (
	mock_v2 "github.com/unicsmcr/hs_auth/mocks/routers/api/v2"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	mock_frontend "github.com/unicsmcr/hs_auth/mocks/routers/frontend"
	"github.com/unicsmcr/hs_auth/testutils"
	"go.uber.org/zap"
)

func Test_RegisterRoutes__should_register_required_routes(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockAPIV2Router := mock_v2.NewMockAPIV2Router(ctrl)
	mockFrontendRouter := mock_frontend.NewMockRouter(ctrl)

	// checking routers get registered on correct paths
	mockFrontendRouter.EXPECT().RegisterRoutes(testutils.RouterGroupMatcher{Path: "/"}).Times(1)
	mockAPIV2Router.EXPECT().RegisterRoutes(testutils.RouterGroupMatcher{Path: "/api/v2"}).Times(1)

	router := NewMainRouter(zap.NewNop(), mockAPIV2Router, mockFrontendRouter)

	w := httptest.NewRecorder()
	_, testServer := gin.CreateTestContext(w)
	router.RegisterRoutes(&testServer.RouterGroup)

	tests := []struct {
		route  string
		method string
	}{}

	for _, tt := range tests {
		t.Run(tt.route, func(t *testing.T) {

			req := httptest.NewRequest(tt.method, tt.route, nil)

			testServer.ServeHTTP(w, req)

			// making sure route is defined
			assert.NotEqual(t, http.StatusNotFound, w.Code)
		})
	}
}
