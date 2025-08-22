package handlers

import (
	"net/http"

	"github.com/PavelMilanov/container-registry/services"
	"github.com/gin-gonic/gin"
)

/*
getRegistry - получение информации о реестрах.

	<name> - название реестра.

	/api/ -вывод всех реестров.
	/api/<name> - вывод всех образов репозитория.
*/
func (h *Handler) getRegistry(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		data, err := services.GetRegistries(h.DB.Sql)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": data})
		return
	}
	data, err := services.GetRepositories(h.DB.Sql, name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

/*
addRegistry -добавление указанного реестра.

	<name> - название реестра.

	/api/<name> - добавление реестра.
*/
func (h *Handler) addRegistry(c *gin.Context) {
	data := c.Param("name")
	if err := services.AddRegistry(data, h.DB.Sql, h.STORAGE); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"err": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{})
}

/*
deleteRegistry -удаление указанного реестра.

	<name> - название реестра.

	/api/<name> - удаляется реестр.
*/
func (h *Handler) deleteRegistry(c *gin.Context) {
	data := c.Param("name")
	if err := services.DeleteRegistry(data, h.DB.Sql, h.STORAGE); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"err": "Ошибка при удалении реестра"})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{})
}

/*
deleteImage -удаление указанного образа.

	<name> - название репозитория.
	<image> - название образа.
	<hash> - хеш образа.

	/api/<name>/<image>?hash=<hash> - удаляется указанный образ.
*/
func (h *Handler) deleteImage(c *gin.Context) {
	name := c.Param("name")
	image := c.Param("image")
	hash := c.Query("hash")
	if hash != "" { // удаляется только образ
		if err := services.DeleteImage(name, image, hash, h.DB.Sql, h.STORAGE); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"err": "Ошибка при удалении образа"})
			return
		}
		c.JSON(http.StatusAccepted, gin.H{})
	} else { // удаляется весь репозиторий
		if err := services.DeleteRepository(name, image, h.DB.Sql, h.STORAGE); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"err": "Ошибка при удалении репозитория"})
			return
		}
		c.JSON(http.StatusAccepted, gin.H{})
	}
}

/*
getImages - получение всех тегов образа.

	<name> - название репозитория.
	<image> - название образа.

	/api/<name>/<image>
*/
func (h *Handler) getImages(c *gin.Context) {
	ImageName := c.Param("image")
	data, err := services.GetImages(ImageName, h.DB.Sql)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"err": err})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "не указан логин или пароль"})
		return
	}
	if req.Password != req.ConfirmPassword {
		c.JSON(http.StatusBadRequest, gin.H{"error": "пароли не совпадают"})
		return
	}
	if err := services.Registration(h.DB.Sql, req.Username, req.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "не указан логин или пароль"})
		return
	}
	token, err := services.Login(h.DB.Sql, h.ENV, req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token})
}

/*
settings - настройки.

	/api/settings - получение настроек.
	/api/settings?tag=<int> - установка количества тегов.
	/api/settings?garbage=true - очистка хранилища.
*/
func (h *Handler) settings(c *gin.Context) {
	if c.Request.Method == "GET" {
		data, err := services.GetSettings(h.DB.Sql, h.STORAGE)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"version":       data.Version,
			"count":         data.Count,
			"total":         data.Total,
			"used":          data.Used,
			"usedToPercent": data.UsedToPercent})
	} else if c.Request.Method == "POST" {
		q := c.Query("garbage")
		t := c.Query("tag")
		if q == "true" {
			h.STORAGE.GarbageCollection()
			c.JSON(http.StatusAccepted, gin.H{"data": "Очистка завершена"})
			return
		}
		if t != "" {
			if err := services.SetCountTag(h.DB.Sql, t); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusAccepted, gin.H{"data": "Настройки сохранены"})
			return
		}
	}
}
