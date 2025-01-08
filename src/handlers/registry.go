package handlers

import (
	"net/http"

	"github.com/PavelMilanov/container-registry/db"
	"github.com/gin-gonic/gin"
)

func (h *Handler) getRepository(c *gin.Context) {
	data := db.GetRegistires(h.DB.Sql)
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) addRepository(c *gin.Context) {
	data := c.Param("name")

	repo := db.Registry{Name: data}
	if err := repo.Add(h.DB.Sql); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": repo})
}

// func (h *Handler) registryView(c *gin.Context) {
// 	registryName := c.Param("name")
// 	registry := db.Registry{Name: registryName}
// 	registry.GetImages(h.DB.Sql)
// 	c.HTML(http.StatusOK, "registry.html", gin.H{
// 		"header":     "Реестры | Container Registry",
// 		"repository": registry,
// 		"pages": []web.Page{
// 			{Name: "Реестры", URL: "/", IsVisible: true},
// 			{Name: "Настройки", URL: "/settings", IsVisible: false},
// 		}})
// }
