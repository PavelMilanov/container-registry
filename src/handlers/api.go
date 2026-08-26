package handlers

import (
	"errors"
	"net/http"

	"github.com/PavelMilanov/container-registry/db"
	registryauth "github.com/PavelMilanov/container-registry/internal/auth"
	"github.com/PavelMilanov/container-registry/services"
	"github.com/labstack/echo/v5"
)

/*
addCloud -добавление указанного пространства.

	{
	 "cloud": "name"
	}

	   /api/cloud/create
*/
func (h *Handler) addCloud(c *echo.Context) error {
	var req struct {
		Cloud string `json:"cloud"`
	}
	if err := c.Bind(&req); err != nil {
		addRequestError(c, err)
		return c.JSON(http.StatusBadRequest, map[string]any{"err": "неверный формат"})
	}
	if err := services.AddCloud(req.Cloud, h.CLOUDS); err != nil {
		addRequestError(c, err)
		return c.JSON(http.StatusBadRequest, map[string]any{"err": err.Error()})
	}
	return c.JSON(http.StatusCreated, map[string]any{"msg": "Пространство создано"})
}

/*
getCloudList -получение списка пространств.

	/api/cloud/list
*/
func (h *Handler) getCloudList(c *echo.Context) error {
	list, err := services.GetCloudList(h.CLOUDS)
	if err != nil {
		addRequestError(c, err)
		return c.JSON(http.StatusBadRequest, map[string]any{"err": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"clouds": list})
}

/*
deleteCloud -удаление указанного пространства.

	{
	 "cloud": "name"
	}

	   /api/cloud/delete
*/
func (h *Handler) deleteCloud(c *echo.Context) error {
	var req struct {
		Cloud string `json:"cloud"`
	}
	if err := c.Bind(&req); err != nil {
		addRequestError(c, err)
		return c.JSON(http.StatusBadRequest, map[string]any{"err": "неверный формат"})
	}
	if err := services.DeleteCloud(req.Cloud, h.CLOUDS); err != nil {
		addRequestError(c, err)
		return c.JSON(http.StatusBadRequest, map[string]any{"err": err.Error()})
	}
	return c.JSON(http.StatusAccepted, map[string]any{"msg": "Пространство удалено"})
}

func (h *Handler) getRepoList(c *echo.Context) error {
	cloud := c.Param("cloud")
	list, err := services.GetRepositoriesList(cloud, h.REPOSITORIES)
	if err != nil {
		addRequestError(c, err)
		return c.JSON(http.StatusBadRequest, map[string]any{"err": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"repositories": list})
}

/*
getImagesList - получение всех образов указанного репозитория.

	<cloud> - название пространства.
	<repository> - название репозитория.

	/api/<cloud>/<repository>
*/
func (h *Handler) getImagesList(c *echo.Context) error {
	cloud := c.Param("cloud")
	repo := c.Param("repository")
	list, err := services.GetImagesList(cloud, repo, h.TAGS)
	if err != nil {
		addRequestError(c, err)
		return c.JSON(http.StatusBadRequest, map[string]any{"err": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"images": list})
}

/*
deleteImage -удаление указанного образа.

	<cloud> - название пространства.
	<repository> - название репозитория.
	<tag> - название образа.

	/api/<cloud>/<repository>?tag=<tag>.
*/
func (h *Handler) deleteRepositoryOrImage(c *echo.Context) error {
	cloud := c.Param("cloud")
	repo := c.Param("repository")
	tag := c.QueryParam("tag")
	if tag != "" {
		if err := services.DeleteImage(cloud, repo, tag, h.TAGS); err != nil {
			addRequestError(c, err)
			return c.JSON(http.StatusBadRequest, map[string]any{"err": "Ошибка при удалении образа"})
		}
		return c.NoContent(http.StatusNoContent)
	} else {
		if err := services.DeleteRepository(cloud, repo, h.REPOSITORIES); err != nil {
			addRequestError(c, err)
			return c.JSON(http.StatusBadRequest, map[string]any{"err": "репозиторий не найден"})
		}
		return c.NoContent(http.StatusNoContent)
	}
}

/*
registration - регистрация.

	<Username> - логин. (обязательное поле)
	<Password> - пароль. (обязательное поле)
	<ConfirmPassword> - подтверждение пароля. (обязательное поле)

	/registration
*/
func (h *Handler) registration(c *echo.Context) error {
	type userRegisterData struct {
		Username        string `json:"username"`
		Password        string `json:"password"`
		ConfirmPassword string `json:"confirmPassword"`
	}
	var req userRegisterData
	if err := c.Bind(&req); err != nil || req.Username == "" ||
		req.Password == "" || req.ConfirmPassword == "" {
		addRequestError(c, err)
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "не указан логин или пароль"})
	}
	if req.Password != req.ConfirmPassword {
		addRequestErrorMessage(c, "пароли не совпадают")
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "пароли не совпадают"})
	}
	if err := h.AUTH.Register(c.Request().Context(), req.Username, req.Password); err != nil {
		addRequestError(c, err)
		switch {
		case errors.Is(err, db.ErrUserExists):
			return c.JSON(http.StatusConflict, map[string]any{"error": "пользователь уже существует"})
		case errors.Is(err, registryauth.ErrEmptyPassword),
			errors.Is(err, registryauth.ErrPasswordTooLong):
			return c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]any{"error": "ошибка регистрации"})
		}
	}
	return c.JSON(http.StatusCreated, map[string]any{"msg": "Пользователь зарегистрирован"})
}

/*
login - авторизация.

	<Username> - логин. (обязательное поле)
	<Password> - пароль. (обязательное поле)

	/login
*/
func (h *Handler) login(c *echo.Context) error {
	type userLoginData struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	var req userLoginData
	if err := c.Bind(&req); err != nil || req.Username == "" || req.Password == "" {
		addRequestError(c, err)
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "не указан логин или пароль"})
	}
	token, err := h.AUTH.Login(
		c.Request().Context(),
		req.Username,
		req.Password,
		nil,
	)
	if err != nil {
		addRequestError(c, err)
		if errors.Is(err, services.ErrInvalidCredentials) {
			return c.JSON(http.StatusUnauthorized, map[string]any{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "ошибка авторизации"})
	}
	return c.JSON(http.StatusOK, map[string]any{"token": token.Value})
}

func (h *Handler) garbageCollection(c *echo.Context) error {
	if err := services.GarbageCollection(h.GARBAGE_COLLECTOR); err != nil {
		addRequestError(c, err)
		return c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusAccepted, map[string]any{"msg": "Очистка завершена"})
}

func (h *Handler) settings(c *echo.Context) error {
	switch c.Request().Method {
	case http.MethodGet:
		count, err := services.GetCountTag(
			c.Request().Context(),
			h.SETTINGS,
		)
		if err != nil {
			addRequestError(c, err)
			return c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, map[string]any{"tagCount": count})
	case http.MethodPost:
		tag := c.QueryParam("tag")
		if tag != "" {
			if err := services.SetCountTag(
				c.Request().Context(),
				h.SETTINGS,
				tag,
			); err != nil {
				addRequestError(c, err)
				return c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
			}
			return c.JSON(http.StatusAccepted, map[string]any{"msg": "Настройки сохранены"})
		} else {
			addRequestErrorMessage(c, "не указано значение tag")
			return c.JSON(http.StatusBadRequest, map[string]any{"error": "не указано значение tag"})
		}
	}
	return c.NoContent(http.StatusMethodNotAllowed)
}

func (h *Handler) deleteOlderTags(c *echo.Context) error {
	if err := services.DeleteOlderTags(
		c.Request().Context(),
		h.SETTINGS,
		h.TAG_PRUNER,
	); err != nil {
		addRequestError(c, err)
		return c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusAccepted, map[string]any{"msg": "Операция завершена успешно"})
}
