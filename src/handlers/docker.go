package handlers

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func (h *Handler) authHandler(c *gin.Context) {
	u, p, ok := c.Request.BasicAuth()
	fmt.Println(u, p, ok)
	// data := c.GetHeader("Authorization")
	// if data == "" || !strings.HasPrefix(data, "Basic ") {
	// 	c.Header("WWW-Authenticate", `Bearer realm="registry"`)
	// 	c.JSON(http.StatusUnauthorized, gin.H{"err": "invalid credentials"})
	// 	return
	// }
	// payload := strings.TrimPrefix(data, "Basic ")
	// decoded, err := base64.StdEncoding.DecodeString(payload)
	// if err != nil {
	// 	logrus.Debug(err)
	// 	return
	// }
	// username := strings.Split(string(decoded), ":")[0]
	// password := strings.Split(string(decoded), ":")[1]
	// Создание JWT токена
	// token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
	// 	"iss": "auth-service",
	// 	"sub": "user",
	// 	"aud": c.Query("service"),
	// 	"exp": time.Now().Add(5 * time.Minute).Unix(),
	// 	"nbf": time.Now().Unix(),
	// 	"iat": time.Now().Unix(),
	// 	"jti": fmt.Sprintf("%d", time.Now().UnixNano()),
	// 	"access": []map[string]interface{}{
	// 		{
	// 			"type":    "repository",
	// 			"name":    c.Query("scope"),
	// 			"actions": []string{"pull", "push"},
	// 		},
	// 	},
	// })

	// tokenString, err := token.SignedString(h.ENV.Server.Jwt)
	// if err != nil {
	// 	c.AbortWithStatus(http.StatusInternalServerError)
	// 	return
	// }

	// c.JSON(http.StatusOK, gin.H{
	// 	"token":        tokenString,
	// 	"access_token": tokenString,
	// 	"expires_in":   300,
	// 	"issued_at":    time.Now().UTC().Format(time.RFC3339),
	// })
	// user := db.User{Name: u, Password: p}
	// if err := user.Login(h.DB.Sql, []byte(h.ENV.Server.Jwt)); err != nil {
	// 	c.Header("WWW-Authenticate", `Basic realm="registry"`)
	// 	c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
	// 	return
	// }
	// c.JSON(http.StatusOK, gin.H{"token": tokenString})
}
