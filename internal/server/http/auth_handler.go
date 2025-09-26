package http

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	authsvc "github.com/zacharykka/prompt-manager/internal/service/auth"
	"github.com/zacharykka/prompt-manager/pkg/httpx"
)

// AuthHandler 处理认证相关请求。
type AuthHandler struct {
	service *authsvc.Service
}

// NewAuthHandler 构造认证处理器。
func NewAuthHandler(service *authsvc.Service) *AuthHandler {
	return &AuthHandler{service: service}
}

// RegisterRoutes 注册认证相关路由。
func (h *AuthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/register", h.Register)
	rg.POST("/login", h.Login)
	rg.POST("/refresh", h.Refresh)
	rg.GET("/github/login", h.GitHubLogin)
	rg.GET("/github/callback", h.GitHubCallback)
}

type registerRequest struct {
	Email    string `json:"email" binding:"required,email,max=255"`
	Password string `json:"password" binding:"required,min=8,max=128"`
	Role     string `json:"role" binding:"omitempty,oneof=admin editor viewer"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email,max=255"`
	Password string `json:"password" binding:"required,min=8,max=128"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Register 创建用户。
func (h *AuthHandler) Register(ctx *gin.Context) {
	var req registerRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		httpx.RespondError(ctx, http.StatusBadRequest, "INVALID_PAYLOAD", err.Error(), nil)
		return
	}

	user, err := h.service.Register(ctx, req.Email, req.Password, req.Role)
	if err != nil {
		h.handleError(ctx, err)
		return
	}

	httpx.RespondOK(ctx, gin.H{"user": user})
}

// Login 校验凭证并返回令牌。
func (h *AuthHandler) Login(ctx *gin.Context) {
	var req loginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		httpx.RespondError(ctx, http.StatusBadRequest, "INVALID_PAYLOAD", err.Error(), nil)
		return
	}

	tokens, user, err := h.service.Login(ctx, req.Email, req.Password)
	if err != nil {
		h.handleError(ctx, err)
		return
	}

	httpx.RespondOK(ctx, gin.H{
		"tokens": tokens,
		"user":   user,
	})
}

// Refresh 使用刷新令牌颁发新访问令牌。
func (h *AuthHandler) Refresh(ctx *gin.Context) {
	var req refreshRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		httpx.RespondError(ctx, http.StatusBadRequest, "INVALID_PAYLOAD", err.Error(), nil)
		return
	}

	tokens, user, err := h.service.Refresh(ctx, req.RefreshToken)
	if err != nil {
		h.handleError(ctx, err)
		return
	}

	httpx.RespondOK(ctx, gin.H{
		"tokens": tokens,
		"user":   user,
	})
}

// GitHubLogin 引导用户跳转至 GitHub 授权页。
func (h *AuthHandler) GitHubLogin(ctx *gin.Context) {
	authorizeURL, err := h.service.GitHubAuthorizeURL(
		ctx.Query("redirect_uri"),
		ctx.Query("response_mode"),
		ctx.Query("client_origin"),
	)
	if err != nil {
		h.handleError(ctx, err)
		return
	}
	ctx.Redirect(http.StatusFound, authorizeURL)
}

// GitHubCallback 处理 GitHub OAuth 回调并返回本地令牌。
func (h *AuthHandler) GitHubCallback(ctx *gin.Context) {
	tokens, user, redirectURI, responseMode, clientOrigin, err := h.service.HandleGitHubCallback(
		ctx.Request.Context(),
		ctx.Query("code"),
		ctx.Query("state"),
	)
	if err != nil {
		h.handleError(ctx, err)
		return
	}

	payload := gin.H{
		"tokens": tokens,
		"user":   user,
	}
	if redirectURI != "" {
		payload["redirect_uri"] = redirectURI
	}

	if responseMode == "web_message" {
		h.respondWebMessage(ctx, payload, redirectURI, clientOrigin)
		return
	}

	httpx.RespondOK(ctx, payload)
}

func (h *AuthHandler) respondWebMessage(ctx *gin.Context, payload gin.H, redirectURI, clientOrigin string) {
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		h.handleError(ctx, fmt.Errorf("marshal web message payload: %w", err))
		return
	}

	encoded := base64.StdEncoding.EncodeToString(jsonBytes)

	// Use clientOrigin as the target origin for postMessage
	targetOrigin := "*"
	if strings.TrimSpace(clientOrigin) != "" {
		targetOrigin = clientOrigin
	} else if redirectURI != "" {
		if parsed, err := url.Parse(redirectURI); err == nil && parsed.Scheme != "" && parsed.Host != "" {
			targetOrigin = fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
		}
	}

	fallbackRedirect := strings.TrimSpace(clientOrigin)

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <title>GitHub 登录完成</title>
</head>
<body>
  <script>
    (function () {
      const encodedPayload = '%s';
      console.log('OAuth callback received, payload length:', encodedPayload.length);

      try {
        const data = JSON.parse(atob(encodedPayload));
        console.log('OAuth payload decoded:', { user: data.user?.email, hasTokens: !!data.tokens });

        console.log('Window opener status:', {
          hasOpener: !!window.opener,
          openerClosed: window.opener ? window.opener.closed : 'no opener',
          windowName: window.name
        });

        if (window.opener && !window.opener.closed) {
          try {
            console.log('Attempting postMessage to opener:', '%s');
            window.opener.postMessage({ source: 'prompt-manager', payload: data }, '%s');
            document.body.innerText = '登录成功，正在返回主窗口...';
            console.log('Posted message to opener window successfully');
            setTimeout(function () {
              window.close();
            }, 1000);
            return;
          } catch (error) {
            console.error('postMessage failed:', error);
            console.log('postMessage failed, trying localStorage fallback');
          }
        } else {
          console.log('No valid opener window found (opener:', !!window.opener, ', closed:', window.opener ? window.opener.closed : 'no opener', '), using localStorage fallback');
        }

        // Force hash redirect for cross-origin scenarios
        console.log('Using hash redirect fallback');

        var clientOrigin = '%s';
        if (clientOrigin && clientOrigin !== '') {
          console.log('Redirecting to client with hash, clientOrigin:', clientOrigin);
          var fallbackTarget = clientOrigin + '/auth/login';
          var redirectURL = fallbackTarget + '#pm_oauth=' + encodeURIComponent(encodedPayload);
          console.log('Final redirect URL:', redirectURL);
          document.body.innerText = '正在返回主窗口...';
          window.location.replace(redirectURL);
        } else {
          console.error('No client origin available for redirect');
          document.body.innerText = '登录完成，请手动返回应用。';
        }
      } catch (error) {
        console.error('Failed to process OAuth callback:', error);
        document.body.innerText = '登录处理失败，请返回应用重试。';
      }
    })();
  </script>
</body>
</html>`, encoded, targetOrigin, targetOrigin, fallbackRedirect)

	ctx.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

func (h *AuthHandler) handleError(ctx *gin.Context, err error) {
	switch err {
	case authsvc.ErrInvalidInput:
		httpx.RespondError(ctx, http.StatusBadRequest, "INVALID_INPUT", err.Error(), nil)
	case authsvc.ErrUserExists:
		httpx.RespondError(ctx, http.StatusConflict, "USER_EXISTS", err.Error(), nil)
	case authsvc.ErrInvalidCredentials:
		httpx.RespondError(ctx, http.StatusUnauthorized, "INVALID_CREDENTIALS", "邮箱或密码错误", nil)
	case authsvc.ErrUserDisabled:
		httpx.RespondError(ctx, http.StatusForbidden, "USER_DISABLED", err.Error(), nil)
	case authsvc.ErrTokenInvalid:
		httpx.RespondError(ctx, http.StatusUnauthorized, "TOKEN_INVALID", err.Error(), nil)
	case authsvc.ErrOAuthDisabled:
		httpx.RespondError(ctx, http.StatusBadRequest, "OAUTH_DISABLED", err.Error(), nil)
	case authsvc.ErrOAuthStateInvalid:
		httpx.RespondError(ctx, http.StatusBadRequest, "OAUTH_STATE_INVALID", err.Error(), nil)
	case authsvc.ErrOAuthExchangeFailed:
		httpx.RespondError(ctx, http.StatusBadGateway, "OAUTH_EXCHANGE_FAILED", err.Error(), nil)
	case authsvc.ErrOAuthEmailMissing:
		httpx.RespondError(ctx, http.StatusBadRequest, "OAUTH_EMAIL_MISSING", err.Error(), nil)
	case authsvc.ErrOAuthOrgUnauthorized:
		httpx.RespondError(ctx, http.StatusForbidden, "OAUTH_ORG_FORBIDDEN", err.Error(), nil)
	default:
		httpx.RespondError(ctx, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
}
