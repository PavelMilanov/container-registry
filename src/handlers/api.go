package handlers

import (
	"net/http"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/PavelMilanov/container-registry/db"
	"github.com/gin-gonic/gin"
)

// getRegistry - получение информации о реестрах.
// /api/registry -вывод всех репозиториев.
// /api/registry/<name> - вывод репозиториев указанного реестра.
func (h *Handler) getRegistry(c *gin.Context) {
	name := c.Param("name")
	if name != "" {
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
	registry := db.Registry{Name: data}
	if err := registry.Add(h.DB.Sql); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": registry})
}

// deleteRegistry -удаление указанного реестра.
func (h *Handler) deleteRegistry(c *gin.Context) {
	data := c.Param("name")
	if err := h.STORAGE.DeleteRegistry(data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
		return
	}
	registy := db.Registry{Name: data}
	if err := registy.Delete(h.DB.Sql); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"data": registy})
}

// deleteRepository -удаление указанного репозитория или образа.
// /api/<registry>/<repository> - удаляется репозиторий.
// /api/<registry>/<repository>?tag=<tag> - удаляется образ.
func (h *Handler) deleteRepository(c *gin.Context) {
	name := c.Param("name")
	image := c.Param("image")
	tag := c.Query("tag")
	if tag != "" { // удаляется только образ
		img := db.Image{Name: image, Tag: tag}
		if err := img.Delete(h.DB.Sql); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
			return
		}
		if err := h.STORAGE.DeleteImage(name, img.Name, img.Tag, img.Hash); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
		}
		c.JSON(http.StatusAccepted, gin.H{"data": img})
	} else { // удаляется весь репозиторий
		repo := db.Repository{Name: image}
		if err := repo.Delete(h.DB.Sql); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
			return
		}
		if err := h.STORAGE.DeleteRepository(name, image); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"data": repo})
	}
}

// getImage -получение образа.
func (h *Handler) getImage(c *gin.Context) {
	ImageName := c.Param("image")
	repo := db.GetRepository(h.DB.Sql, ImageName)
	data := db.GetImageTags(h.DB.Sql, repo.ID, ImageName)
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
		if q == "true" {
			h.STORAGE.GarbageCollection()
			c.JSON(http.StatusAccepted, gin.H{"data": "Очистка завершена"})
			return
		}
	}
}
