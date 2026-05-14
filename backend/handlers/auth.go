package handlers

import (
	"context"
	"encoding/json"
	"net/http"
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

func GoogleLogin(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		url := oauthConfig(cfg).AuthCodeURL("state-token", oauth2.AccessTypeOnline)
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

		frontendURL := cfg.FrontendURL
		if frontendURL == "" {
			frontendURL = "http://localhost:5173"
		}
		c.Redirect(http.StatusTemporaryRedirect, frontendURL+"/auth/callback?token="+signed)
	}
}
