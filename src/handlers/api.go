package handlers

import (
	"net/http"

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
	registy.Add(h.DB.Sql)
	c.JSON(http.StatusCreated, gin.H{"data": registy})
}

func (h *Handler) deleteRegistry(c *gin.Context) {
	data := c.Param("name")
	registy := db.Registry{Name: data}
	registy.Delete(h.DB.Sql)
	c.JSON(http.StatusAccepted, gin.H{"data": "success"})
}

func (h *Handler) getRepository(c *gin.Context) {
	data := db.GetRepositories(h.DB.Sql)
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) getImage(c *gin.Context) {
	ImageName := c.Param("image")
	repo := db.GetRepository(h.DB.Sql, ImageName)
	data := db.GetImageTags(h.DB.Sql, repo.ID, ImageName)
	c.JSON(http.StatusOK, gin.H{"data": data})
}
