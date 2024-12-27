package handlers

import (
	"net/http"

	"github.com/PavelMilanov/container-registry/db"
	"github.com/PavelMilanov/container-registry/web"
	"github.com/gin-gonic/gin"
)

func (h *Handler) registryView(c *gin.Context) {
	c.HTML(http.StatusOK, "registry.html", gin.H{
		"header": "Реестры | Container Registry",
		"repos":  db.GetRegistires(h.DB.Sql),
		"pages": []web.Page{
			{Name: "Реестры", URL: "/", IsVisible: true},
			{Name: "Настройки", URL: "/settings", IsVisible: false},
		}})
}

func (h *Handler) addRegistryView(c *gin.Context) {
	var data web.Repository
	if err := c.ShouldBind(&data); err != nil {
		return
	}
	registry := db.Registry{Name: data.Name}
	if err := registry.Add(h.DB.Sql); err != nil {
		c.HTML(http.StatusBadRequest, "registry.html", gin.H{"error": err.Error()})
		return
	}
	c.HTML(http.StatusOK, "registry.html", gin.H{
		"header": "Реестры | Container Registry",
		"repos":  db.GetRegistires(h.DB.Sql),
		"pages": []web.Page{
			{Name: "Реестры", URL: "/", IsVisible: true},
			{Name: "Настройки", URL: "/settings", IsVisible: false},
		}})
}
