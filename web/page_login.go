package web

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// getJWTSecret генерирует динамический секретный ключ на основе логина и пароля
func (s *Server) getJWTSecret() []byte {
	username := s.core.GetConfig().Web.Username
	password := s.core.GetConfig().Web.Password
	return []byte(username + ":" + password + ":jwt_secure_salt_2026")
}

// generateJWT генерирует JWT-токен на 30 дней
func (s *Server) generateJWT() (string, error) {
	username := s.core.GetConfig().Web.Username
	secret := s.getJWTSecret()

	// Наполняем токен полезной нагрузкой (claims)
	claims := jwt.MapClaims{
		"username": username,
		"exp":      time.Now().Add(30 * 24 * time.Hour).Unix(), // время жизни 30 дней
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// AuthMiddleware защищает роуты с помощью проверки JWT-токена
func (s *Server) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := s.core.GetConfig().Web.Username
		password := s.core.GetConfig().Web.Password

		// Если в конфигурации авторизация отключена, пропускаем всех без проверок
		if username == "" || password == "" {
			c.Next()
			return
		}

		cookie, err := c.Cookie("bambu_token")
		if err != nil {
			s.unauthorized(c)
			return
		}

		// Декодируем и верифицируем подпись JWT
		secret := s.getJWTSecret()
		token, err := jwt.Parse(cookie, func(token *jwt.Token) (interface{}, error) {
			// Проверяем метод подписи
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return secret, nil
		})

		if err != nil || !token.Valid {
			s.unauthorized(c)
			return
		}

		c.Next()
	}
}

// unauthorized обрабатывает ошибки авторизации в зависимости от типа запроса
func (s *Server) unauthorized(c *gin.Context) {
	// Если это Ajax-запрос или POST формы, отдаем 401 ошибку
	if c.Request.Header.Get("X-Requested-With") == "XMLHttpRequest" || c.Request.Method == "POST" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	// Для обычных страниц перенаправляем на страницу входа
	c.Redirect(http.StatusSeeOther, "/login")
	c.Abort()
}

// LoginGetHandler отображает внешний файл-шаблон login.go.html
func (s *Server) LoginGetHandler(c *gin.Context) {
	// Если авторизация отключена
	if s.core.GetConfig().Web.Username == "" || s.core.GetConfig().Web.Password == "" {
		c.Redirect(http.StatusSeeOther, "/")
		return
	}

	// Если пользователь уже авторизован по JWT
	cookie, err := c.Cookie("bambu_token")
	if err == nil {
		secret := s.getJWTSecret()
		token, err := jwt.Parse(cookie, func(token *jwt.Token) (interface{}, error) {
			return secret, nil
		})
		if err == nil && token.Valid {
			c.Redirect(http.StatusSeeOther, "/")
			return
		}
	}

	// Рендерим внешний шаблон
	c.HTML(http.StatusOK, "login.go.html", gin.H{})
}

// LoginPostHandler обрабатывает POST форму входа
func (s *Server) LoginPostHandler(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	expectedUser := s.core.GetConfig().Web.Username
	expectedPass := s.core.GetConfig().Web.Password

	if username == expectedUser && password == expectedPass {
		tokenString, err := s.generateJWT()
		if err != nil {
			c.HTML(http.StatusInternalServerError, "login.go.html", gin.H{
				"Error": "Ошибка генерации токена авторизации",
			})
			return
		}

		// Записываем подписанный JWT-токен в Cookies на 30 дней
		c.SetCookie("bambu_token", tokenString, 2592000, "/", "", false, true)
		c.Redirect(http.StatusSeeOther, "/")
		return
	}

	c.HTML(http.StatusUnauthorized, "login.go.html", gin.H{
		"Error": "Неверное имя пользователя или пароль",
	})
}

// LogoutHandler стирает сессию
func (s *Server) LogoutHandler(c *gin.Context) {
	c.SetCookie("bambu_token", "", -1, "/", "", false, true)
	c.Redirect(http.StatusSeeOther, "/login")
}
