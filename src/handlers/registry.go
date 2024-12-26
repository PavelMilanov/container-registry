package handlers

import (
	"net/http"

	"github.com/PavelMilanov/container-registry/web"
	"github.com/gin-gonic/gin"
)

func (h *Handler) registryView(c *gin.Context) {
	c.HTML(http.StatusOK, "registry.html", gin.H{
		// "header":          "Главная | PgBackup",
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
