package handlers

import (
	"net/http"

	"github.com/PavelMilanov/container-registry/services"
	"github.com/gin-gonic/gin"
)

/*
addCloud -добавление указанного пространства.

	{
	 "cloud": "name"
	}

	   /api/cloud/create
*/
func (h *Handler) addCloud(c *gin.Context) {
	var req struct {
		Cloud string `json:"cloud"`
	}
	if err := c.BindJSON(&req); err != nil {
		addRequestError(c, err)
		c.JSON(http.StatusBadRequest, gin.H{"err": "неверный формат"})
		return
	}
	if err := services.AddCloud(req.Cloud, h.STORAGE); err != nil {
		addRequestError(c, err)
		c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"msg": "Пространство создано"})
}

/*
getCloudList -получение списка пространств.

	/api/cloud/list
*/
func (h *Handler) getCloudList(c *gin.Context) {
	list, err := services.GetCloudList(h.STORAGE)
	if err != nil {
		addRequestError(c, err)
		c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"clouds": list})
}

/*
deleteCloud -удаление указанного пространства.

	{
	 "cloud": "name"
	}

	   /api/cloud/delete
*/
func (h *Handler) deleteCloud(c *gin.Context) {
	var req struct {
		Cloud string `json:"cloud"`
	}
	if err := c.BindJSON(&req); err != nil {
		addRequestError(c, err)
		c.JSON(http.StatusBadRequest, gin.H{"err": "неверный формат"})
		return
	}
	if err := services.DeleteCloud(req.Cloud, h.STORAGE); err != nil {
		addRequestError(c, err)
		c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"msg": "Пространство удалено"})
}

func (h *Handler) getRepoList(c *gin.Context) {
	cloud := c.Param("cloud")
	list, err := services.GetRepositoriesList(cloud, h.STORAGE)
	if err != nil {
		addRequestError(c, err)
		c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"repositories": list})
}

/*
getImagesList - получение всех образов указанного репозитория.

	<cloud> - название пространства.
	<repository> - название репозитория.

	/api/<cloud>/<repository>
*/
func (h *Handler) getImagesList(c *gin.Context) {
	cloud := c.Param("cloud")
	repo := c.Param("repository")
	list, err := services.GetImagesList(cloud, repo, h.STORAGE)
	if err != nil {
		addRequestError(c, err)
		c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"images": list})
}

/*
deleteImage -удаление указанного образа.

	<cloud> - название пространства.
	<repository> - название репозитория.
	<tag> - название образа.

	/api/<cloud>/<repository>?tag=<tag>.
*/
func (h *Handler) deleteRepositoryOrImage(c *gin.Context) {
	cloud := c.Param("cloud")
	repo := c.Param("repository")
	tag := c.Query("tag")
	if tag != "" {
		if err := services.DeleteImage(cloud, repo, tag, h.STORAGE); err != nil {
			addRequestError(c, err)
			c.JSON(http.StatusBadRequest, gin.H{"err": "Ошибка при удалении образа"})
			return
		}
		c.JSON(http.StatusNoContent, gin.H{"msg": "Образ успешно удален"})
	} else {
		if err := services.DeleteRepository(cloud, repo, h.STORAGE); err != nil {
			addRequestError(c, err)
			c.JSON(http.StatusBadRequest, gin.H{"err": "репозиторий не найден"})
			return
		}
		c.JSON(http.StatusNoContent, gin.H{"msg": "Репозиторий успешно удален"})
	}
}

/*
registration - регистрация.

	<Username> - логин. (обязательное поле)
	<Password> - пароль. (обязательное поле)
	<ConfirmPassword> - подтверждение пароля. (обязательное поле)

	/registration
*/
func (h *Handler) registration(c *gin.Context) {
	type userRegisterData struct {
		Username        string `json:"username" binding:"required"`
		Password        string `json:"password" binding:"required"`
		ConfirmPassword string `json:"confirmPassword" binding:"required"`
	}
	var req userRegisterData
	if err := c.BindJSON(&req); err != nil {
		addRequestError(c, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "не указан логин или пароль"})
		return
	}
	if req.Password != req.ConfirmPassword {
		addRequestErrorMessage(c, "пароли не совпадают")
		c.JSON(http.StatusBadRequest, gin.H{"error": "пароли не совпадают"})
		return
	}
	if err := services.Registration(h.DB.Sql, req.Username, req.Password); err != nil {
		addRequestError(c, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"msg": "Пользователь зарегистрирован"})
}

/*
login - авторизация.

	<Username> - логин. (обязательное поле)
	<Password> - пароль. (обязательное поле)

	/login
*/
func (h *Handler) login(c *gin.Context) {
	type userLoginData struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	var req userLoginData
	if err := c.BindJSON(&req); err != nil {
		addRequestError(c, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "не указан логин или пароль"})
		return
	}
	token, err := services.Login(h.DB.Sql, h.ENV, req.Username, req.Password)
	if err != nil {
		addRequestError(c, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token})
}

func (h *Handler) garbageCollection(c *gin.Context) {
	if err := services.GarbageCollection(h.STORAGE); err != nil {
		addRequestError(c, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"msg": "Очистка завершена"})
}

func (h *Handler) settings(c *gin.Context) {
	switch c.Request.Method {
	case http.MethodGet:
		count, err := services.GetCountTag(h.DB.Sql)
		if err != nil {
			addRequestError(c, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"tagCount": count})
	case http.MethodPost:
		tag := c.Query("tag")
		if tag != "" {
			if err := services.SetCountTag(h.DB.Sql, tag); err != nil {
				addRequestError(c, err)
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusAccepted, gin.H{"msg": "Настройки сохранены"})
		} else {
			addRequestErrorMessage(c, "не указано значение tag")
			c.JSON(http.StatusBadRequest, gin.H{"error": "не указано значение tag"})
		}
	}
}

func (h *Handler) deleteOlderTags(c *gin.Context) {
	if err := services.DeleteOlderTags(h.DB.Sql, h.STORAGE); err != nil {
		addRequestError(c, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"msg": "Операция завершена успешно"})
}
