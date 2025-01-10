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

	repo := db.Registry{Name: data}
	if err := repo.Add(h.DB.Sql); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": repo})
}

func (h *Handler) getRepositoryTags(c *gin.Context) {
	data := db.GetImages(h.DB.Sql)
	c.JSON(http.StatusOK, gin.H{"data": data})
}
