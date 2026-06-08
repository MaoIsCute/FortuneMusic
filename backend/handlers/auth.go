package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"fortune-tracker/config"
	"fortune-tracker/db"
	"fortune-tracker/middleware"
	"fortune-tracker/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func oauthConfig(cfg *config.Config) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.AppURL + "/auth/google/callback",
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
}

// generateState builds a state string: <nonce>:<HMAC-SHA256(nonce, secret)>
func generateState(secret string) string {
	nonce := make([]byte, 16)
	rand.Read(nonce)
	nonceHex := hex.EncodeToString(nonce)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(nonceHex))
	return nonceHex + ":" + hex.EncodeToString(mac.Sum(nil))
}

// verifyState checks that the state's HMAC is valid.
func verifyState(state, secret string) bool {
	parts := strings.SplitN(state, ":", 2)
	if len(parts) != 2 {
		return false
	}
	nonceHex, sig := parts[0], parts[1]
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(nonceHex))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(sig), []byte(expected))
}

// authCode holds a short-lived JWT for the code exchange flow.
type authCode struct {
	token     string
	userID    uint
	expiresAt time.Time
}

var authCodeStore sync.Map

func init() {
	go func() {
		for range time.Tick(time.Minute) {
			authCodeStore.Range(func(k, v any) bool {
				if time.Now().After(v.(authCode).expiresAt) {
					authCodeStore.Delete(k)
				}
				return true
			})
		}
	}()
}

func GoogleLogin(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		state := generateState(cfg.JWTSecret)
		url := oauthConfig(cfg).AuthCodeURL(state, oauth2.AccessTypeOnline)
		c.Redirect(http.StatusTemporaryRedirect, url)
	}
}

type googleUser struct {
	ID    string `json:"sub"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func GoogleCallback(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		frontendURL := cfg.FrontendURL
		if frontendURL == "" {
			frontendURL = "http://localhost:5173"
		}

		if c.Query("error") != "" {
			c.Redirect(http.StatusTemporaryRedirect, frontendURL+"/?error=cancelled")
			return
		}

		if !verifyState(c.Query("state"), cfg.JWTSecret) {
			c.Redirect(http.StatusTemporaryRedirect, frontendURL+"/?error=cancelled")
			return
		}

		oc := oauthConfig(cfg)
		token, err := oc.Exchange(context.Background(), c.Query("code"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "token exchange failed"})
			return
		}

		resp, err := oc.Client(context.Background(), token).Get("https://www.googleapis.com/oauth2/v3/userinfo")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user info"})
			return
		}
		defer resp.Body.Close()

		var gu googleUser
		if err := json.NewDecoder(resp.Body).Decode(&gu); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode user info"})
			return
		}

		var user models.User
		if err := db.DB.Where("google_id = ?", gu.ID).First(&user).Error; err != nil {
			user = models.User{GoogleID: gu.ID, Email: gu.Email, Name: gu.Name}
			db.DB.Create(&user)
		}

		claims := &middleware.Claims{
			UserID: user.ID,
			Email:  user.Email,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			},
		}
		signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(cfg.JWTSecret))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to sign token"})
			return
		}

		// Issue a short-lived opaque code instead of exposing JWT in the URL.
		codeBytes := make([]byte, 24)
		rand.Read(codeBytes)
		code := hex.EncodeToString(codeBytes)
		authCodeStore.Store(code, authCode{token: signed, userID: user.ID, expiresAt: time.Now().Add(60 * time.Second)})

		c.Redirect(http.StatusTemporaryRedirect, frontendURL+"/auth/callback?code="+code)
	}
}

// ExchangeToken exchanges a one-time code for a JWT + refresh token.
func ExchangeToken(c *gin.Context) {
	var req struct {
		Code string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing code"})
		return
	}

	val, ok := authCodeStore.LoadAndDelete(req.Code)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired code"})
		return
	}

	ac := val.(authCode)
	if time.Now().After(ac.expiresAt) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "code expired"})
		return
	}

	rtBytes := make([]byte, 32)
	rand.Read(rtBytes)
	rt := hex.EncodeToString(rtBytes)
	db.DB.Model(&models.User{}).Where("id = ?", ac.userID).Updates(map[string]interface{}{
		"refresh_token":        rt,
		"refresh_token_expiry": time.Now().Add(30 * 24 * time.Hour),
	})

	c.JSON(http.StatusOK, gin.H{"token": ac.token, "refresh_token": rt})
}

// RefreshJWT issues a new JWT given a valid refresh token.
func RefreshJWT(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing refresh_token"})
			return
		}

		var user models.User
		if err := db.DB.Where("refresh_token = ?", req.RefreshToken).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
			return
		}
		if time.Now().After(user.RefreshTokenExpiry) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token expired"})
			return
		}

		claims := &middleware.Claims{
			UserID: user.ID,
			Email:  user.Email,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			},
		}
		signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(cfg.JWTSecret))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to sign token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"token": signed})
	}
}
