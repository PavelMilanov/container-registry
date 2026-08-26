package handlers

import (
	"errors"
	"net/http"

	"github.com/PavelMilanov/container-registry/db"
	registryauth "github.com/PavelMilanov/container-registry/internal/auth"
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
	if err := services.AddCloud(req.Cloud, h.CLOUDS); err != nil {
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
	list, err := services.GetCloudList(h.CLOUDS)
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
	if err := services.DeleteCloud(req.Cloud, h.CLOUDS); err != nil {
		addRequestError(c, err)
		c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"msg": "Пространство удалено"})
}

func (h *Handler) getRepoList(c *gin.Context) {
	cloud := c.Param("cloud")
	list, err := services.GetRepositoriesList(cloud, h.REPOSITORIES)
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
	list, err := services.GetImagesList(cloud, repo, h.TAGS)
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
		if err := services.DeleteImage(cloud, repo, tag, h.TAGS); err != nil {
			addRequestError(c, err)
			c.JSON(http.StatusBadRequest, gin.H{"err": "Ошибка при удалении образа"})
			return
		}
		c.JSON(http.StatusNoContent, gin.H{"msg": "Образ успешно удален"})
	} else {
		if err := services.DeleteRepository(cloud, repo, h.REPOSITORIES); err != nil {
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
	if err := h.AUTH.Register(c.Request.Context(), req.Username, req.Password); err != nil {
		addRequestError(c, err)
		switch {
		case errors.Is(err, db.ErrUserExists):
			c.JSON(http.StatusConflict, gin.H{"error": "пользователь уже существует"})
		case errors.Is(err, registryauth.ErrEmptyPassword),
			errors.Is(err, registryauth.ErrPasswordTooLong):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка регистрации"})
		}
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
	token, err := h.AUTH.Login(
		c.Request.Context(),
		req.Username,
		req.Password,
		nil,
	)
	if err != nil {
		addRequestError(c, err)
		if errors.Is(err, services.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка авторизации"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token.Value})
}

func (h *Handler) garbageCollection(c *gin.Context) {
	if err := services.GarbageCollection(h.GARBAGE_COLLECTOR); err != nil {
		addRequestError(c, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"msg": "Очистка завершена"})
}

func (h *Handler) settings(c *gin.Context) {
	switch c.Request.Method {
	case http.MethodGet:
		count, err := services.GetCountTag(
			c.Request.Context(),
			h.SETTINGS,
		)
		if err != nil {
			addRequestError(c, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"tagCount": count})
	case http.MethodPost:
		tag := c.Query("tag")
		if tag != "" {
			if err := services.SetCountTag(
				c.Request.Context(),
				h.SETTINGS,
				tag,
			); err != nil {
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
	if err := services.DeleteOlderTags(
		c.Request.Context(),
		h.SETTINGS,
		h.TAG_PRUNER,
	); err != nil {
		addRequestError(c, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"msg": "Операция завершена успешно"})
}
