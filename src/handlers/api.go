package handlers

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/PavelMilanov/container-registry/db"
	"github.com/gin-gonic/gin"
)

func (h *Handler) getRegistry(c *gin.Context) {
	data := db.GetRegistires(h.DB.Sql)
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) addRegistry(c *gin.Context) {
	data := c.Param("name")
	registy := db.Registry{Name: data}
	if err := registy.Add(h.DB.Sql); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": registy})
}

func (h *Handler) deleteRegistry(c *gin.Context) {
	data := c.Param("name")
	if err := os.RemoveAll(filepath.Join(h.STORAGE.ManifestPath, data)); err != nil {
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

func (h *Handler) getRepository(c *gin.Context) {
	data := db.GetRepositories(h.DB.Sql)
	c.JSON(http.StatusOK, gin.H{"data": data})
}

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
		os.Remove(filepath.Join(h.STORAGE.ManifestPath, name, img.Name, "tags", img.Tag))
		os.Remove(filepath.Join(h.STORAGE.ManifestPath, name, img.Name, img.Hash))
		c.JSON(http.StatusAccepted, gin.H{"data": img})
	} else { // удаляется весь репозиторий
		repo := db.Repository{Name: image}
		if err := repo.Delete(h.DB.Sql); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
			return
		}
		os.RemoveAll(filepath.Join(h.STORAGE.ManifestPath, name, image))
		c.JSON(http.StatusAccepted, gin.H{"data": repo})
	}
}

func (h *Handler) getImage(c *gin.Context) {
	ImageName := c.Param("image")
	repo := db.GetRepository(h.DB.Sql, ImageName)
	data := db.GetImageTags(h.DB.Sql, repo.ID, ImageName)
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) registration(c *gin.Context) {
	type userRegisterData struct {
		Username        string `json:"username" binding:"required"`
		Password        string `json:"password" binding:"required"`
		ConfirmPassword string `json:"confirmPassword" binding:"required"`
	}
	var req userRegisterData
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
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
	c.JSON(http.StatusCreated, gin.H{"data": user})
}

func (h *Handler) login(c *gin.Context) {

	type userLoginData struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	var req userLoginData
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}
	user := db.User{Name: req.Username, Password: req.Password}
	if err := user.Login(h.DB.Sql); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": user.Token})
}
