// Package ginoidc adapts seckit/oidc to Gin routers and middlewares.
package ginoidc

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/niudaii/goutil/seckit/oidc"
)

// Register mounts login, callback, and logout handlers on router.
func Register(router *gin.Engine, client *oidc.Client) {
	if router == nil || client == nil {
		return
	}
	router.GET(client.LoginPath(), gin.WrapF(client.HandleLogin))
	if !client.RootCallback() {
		router.GET(client.CallbackPath(), gin.WrapF(client.HandleCallback))
	}
	router.POST(client.LogoutPath(), SameOrigin(client), Logout(client))
}

// NoStore applies no-store and no-referrer headers on Gin responses.
func NoStore(c *gin.Context) {
	oidc.SetNoStoreHeaders(c.Writer)
}

// Subject returns the authenticated subject stored on the Gin context or session cookie.
func Subject(client *oidc.Client, c *gin.Context) string {
	if value, exists := c.Get(oidc.SubjectContextKey); exists {
		if sub, ok := value.(string); ok && sub != "" {
			return sub
		}
	}
	if client == nil {
		return ""
	}
	return client.Subject(c.Request)
}

// SubjectFromContext returns the subject recorded by Require.
func SubjectFromContext(c *gin.Context) string {
	value, exists := c.Get(oidc.SubjectContextKey)
	if !exists {
		return ""
	}
	sub, _ := value.(string)
	return sub
}

// StartHomeLogin redirects unauthenticated GET/HEAD home requests to OIDC login
// unless login_error is already present.
func StartHomeLogin(client *oidc.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		NoStore(c)
		if client != nil && client.Subject(c.Request) == "" {
			switch c.Query("login_error") {
			case "denied", "failed", "expired":
			default:
				c.Abort()
				client.HandleLogin(c.Writer, c.Request)
				return
			}
		}
		c.Next()
	}
}

// SameOrigin enforces client.OriginAllowed on mutating requests.
func SameOrigin(client *oidc.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		NoStore(c)
		if client != nil && !client.OriginAllowed(c.Request) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    http.StatusForbidden,
				"message": "不允许跨站请求",
			})
			return
		}
		c.Next()
	}
}

// Require verifies the OIDC session, stores SubjectContextKey, and enforces SameOrigin.
func Require(client *oidc.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		NoStore(c)
		if client == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    http.StatusUnauthorized,
				"message": "请先登录",
			})
			return
		}
		subject := client.Subject(c.Request)
		if subject == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    http.StatusUnauthorized,
				"message": "请先登录",
			})
			return
		}
		c.Set(oidc.SubjectContextKey, subject)
		SameOrigin(client)(c)
	}
}

// Status returns standard OIDC authentication status JSON.
func Status(client *oidc.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		NoStore(c)
		var subject string
		if client != nil {
			subject = client.Subject(c.Request)
		}
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"enabled":       client != nil,
				"authenticated": client == nil || subject != "",
				"subject":       subject,
			},
		})
	}
}

// Logout revokes the browser session and clears cookies.
func Logout(client *oidc.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		NoStore(c)
		if client != nil {
			client.RevokeRequest(c.Writer, c.Request)
		}
		c.JSON(http.StatusOK, gin.H{"code": http.StatusOK})
	}
}

// RequireAPIKeyOrSession accepts either a valid X-API-Key header or an authenticated OIDC session.
// If X-API-Key is supplied but invalid, it fails closed with 401 without falling back to session auth.
func RequireAPIKeyOrSession(client *oidc.Client, apiKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		provided, valid := oidc.ValidateAPIKey(c.Request, apiKey)
		if provided {
			NoStore(c)
			if valid {
				c.Next()
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    http.StatusUnauthorized,
				"message": "API Key 无效",
			})
			return
		}
		if client != nil {
			Require(client)(c)
			return
		}
		if apiKey != "" {
			NoStore(c)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    http.StatusUnauthorized,
				"message": "无效或缺少 API Key",
			})
			return
		}
		c.Next()
	}
}
