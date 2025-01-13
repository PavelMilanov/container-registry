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
	name := c.Param("image")
	repo := db.Repository{Name: name}
	if err := repo.Delete(h.DB.Sql); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"data": repo})
}

func (h *Handler) getImage(c *gin.Context) {
	ImageName := c.Param("image")
	repo := db.GetRepository(h.DB.Sql, ImageName)
	data := db.GetImageTags(h.DB.Sql, repo.ID, ImageName)
	c.JSON(http.StatusOK, gin.H{"data": data})
}
