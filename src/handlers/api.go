package handlers

import (
	"net/http"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/PavelMilanov/container-registry/db"
	"github.com/PavelMilanov/container-registry/services"
	"github.com/gin-gonic/gin"
)

// getRegistry - получение информации о реестрах.
// /api/registry -вывод всех репозиториев.
// /api/registry/<name> - вывод репозиториев указанного реестра.
func (h *Handler) getRegistry(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		data := db.GetRegistires(h.DB.Sql)
		c.JSON(http.StatusOK, gin.H{"data": data})
		return
	}
	var registry db.Registry
	if err := registry.GetRepositories(h.DB.Sql, name); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": registry.Repositories})
}

// addRegistry -добавление нового реестра.
func (h *Handler) addRegistry(c *gin.Context) {
	data := c.Param("name")
	// registry := db.Registry{Name: data}
	if err := services.AddRegistry(data, h.DB.Sql); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"err": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{})
}

// deleteRegistry -удаление указанного реестра.
func (h *Handler) deleteRegistry(c *gin.Context) {
	data := c.Param("name")
	if err := services.DeleteRegistry(data, h.DB.Sql, h.STORAGE); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"err": err.Error()})
		return
	}
	// if err := h.STORAGE.DeleteRegistry(data); err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
	// 	return
	// }
	// registy := db.Registry{Name: data}
	// if err := registy.Delete(h.DB.Sql); err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
	// 	return
	// }
	c.JSON(http.StatusAccepted, gin.H{})
}

// deleteRepository -удаление указанного репозитория или образа.
// /api/<registry>/<repository> - удаляется репозиторий.
// /api/<registry>/<repository>?tag=<tag> - удаляется образ.
func (h *Handler) deleteImage(c *gin.Context) {
	name := c.Param("name")
	image := c.Param("image")
	tag := c.Query("tag")
	if tag != "" { // удаляется только образ
		if err := services.DeleteImage(name, image, tag, h.DB.Sql, h.STORAGE); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"err": err.Error()})
			return
		}
		c.JSON(http.StatusAccepted, gin.H{})
	} else { // удаляется весь репозиторий
		if err := services.DeleteRepository(name, image, h.DB.Sql, h.STORAGE); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"err": err.Error()})
			return
		}
		c.JSON(http.StatusAccepted, gin.H{})
	}
}

// getImage -получение образа.
func (h *Handler) getImage(c *gin.Context) {
	ImageName := c.Param("image")
	data := services.GetImages(ImageName, h.DB.Sql)
	// repo := db.GetRepository(h.DB.Sql, ImageName)
	// data := db.GetImageTags(h.DB.Sql, repo.ID, ImageName)
	c.JSON(http.StatusOK, gin.H{"data": data})
}

// registration - регистрация.
func (h *Handler) registration(c *gin.Context) {
	type userRegisterData struct {
		Username        string `json:"username" binding:"required"`
		Password        string `json:"password" binding:"required"`
		ConfirmPassword string `json:"confirmPassword" binding:"required"`
	}
	var req userRegisterData
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "не указан логин или пароль"})
		return
	}
	if req.Password != req.ConfirmPassword {
		c.JSON(http.StatusBadRequest, gin.H{"error": "пароли не совпадают"})
		return
	}
	user := db.User{Name: req.Username, Password: req.Password}
	if err := user.Add(h.DB.Sql); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{})
}

// login - авторизация.
func (h *Handler) login(c *gin.Context) {
	type userLoginData struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	var req userLoginData
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "не указан логин или пароль"})
		return
	}
	user := db.User{Name: req.Username, Password: req.Password}
	if err := user.Login(h.DB.Sql, []byte(h.ENV.Server.Jwt)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": user.Token})
}

func (h *Handler) settings(c *gin.Context) {
	if c.Request.Method == "GET" {
		c.JSON(http.StatusOK, gin.H{"version": config.VERSION})
	} else if c.Request.Method == "POST" {
		q := c.Query("garbage")
		t := c.Query("tag")
		if q == "true" {
			h.STORAGE.GarbageCollection()
			c.JSON(http.StatusAccepted, gin.H{"data": "Очистка завершена"})
			return
		}
		if t != "" {
			c.JSON(http.StatusOK, gin.H{"data": "ping"})
			return
		}
	}
}
