package handlers

import (
	"net/http"

	"github.com/PavelMilanov/container-registry/web"
	"github.com/gin-gonic/gin"
)

func (h *Handler) registryView(c *gin.Context) {
	c.HTML(http.StatusOK, "registry.html", gin.H{
		"header": "Реестры | Container Registry",
		// "storage":         storage,
		// "count":           count,
		// "system":          system,
		// "backups":         lastBackups,
		// "backups_count":   countBackups,
		// "schedules_count": countSchedules,
		"pages": []web.Page{
			// {Name: "Главная", URL: "/", IsVisible: false},
			{Name: "Реестры", URL: "/", IsVisible: true},
			{Name: "Настройки", URL: "/settings", IsVisible: false},
		}})
}

func (h *Handler) addRepositoryView(c *gin.Context) {
	var data web.Repository
	if err := c.ShouldBind(&data); err != nil {
		return
	}
	c.HTML(http.StatusOK, "registry.html", gin.H{
		"header": "Реестры | Container Registry",
		// "storage":         storage,
		// "count":           count,
		// "system":          system,
		"repos": []web.Repository{data},
		// "backups_count":   countBackups,
		// "schedules_count": countSchedules,
		"pages": []web.Page{
			// {Name: "Главная", URL: "/", IsVisible: false},
			{Name: "Реестры", URL: "/", IsVisible: true},
			{Name: "Настройки", URL: "/settings", IsVisible: false},
		}})
}
