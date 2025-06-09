package handlers

import (
	"net/http"
	"strconv"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/PavelMilanov/container-registry/db"
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
		data := db.GetRegistires(h.DB.Sql)
		c.JSON(http.StatusOK, gin.H{"data": data})
		return
	}
	var registry db.Registry
	if err := registry.GetRepositories(h.DB.Sql, name); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": registry.Repositories})
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
		c.JSON(http.StatusForbidden, gin.H{"err": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{})
}

/*
deleteImage -удаление указанного образа.

	<name> - название репозитория.
	<image> - название образа.
	<tag> - тег образа.

	/api/<name>/<image>?tag=<tag> - удаляется указанный образ.
*/
func (h *Handler) deleteImage(c *gin.Context) {
	name := c.Param("name")
	image := c.Param("image")
	tag := c.Query("tag")
	if tag != "" { // удаляется только образ
		if err := services.DeleteImage(name, image, tag, h.DB.Sql, h.STORAGE); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"err": err.Error()})
			return
		}
		c.JSON(http.StatusAccepted, gin.H{})
	} else { // удаляется весь репозиторий
		if err := services.DeleteRepository(name, image, h.DB.Sql, h.STORAGE); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"err": err.Error()})
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
	data := services.GetImages(ImageName, h.DB.Sql)
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
	user := db.User{Name: req.Username, Password: req.Password}
	if err := user.Add(h.DB.Sql); err != nil {
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
	user := db.User{Name: req.Username, Password: req.Password}
	if err := user.Login(h.DB.Sql, h.ENV); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": user.Token})
}

/*
settings - настройки.

	/api/settings - получение настроек.
	/api/settings?tag=<int> - установка количества тегов.
	/api/settings?garbage=true - очистка хранилища.
*/
func (h *Handler) settings(c *gin.Context) {
	if c.Request.Method == "GET" {
		count, _ := db.GetCountTag(h.DB.Sql)
		c.JSON(http.StatusOK, gin.H{"version": config.VERSION, "count": count})
	} else if c.Request.Method == "POST" {
		q := c.Query("garbage")
		t := c.Query("tag")
		if q == "true" {
			h.STORAGE.GarbageCollection()
			c.JSON(http.StatusAccepted, gin.H{"data": "Очистка завершена"})
			return
		}
		if t != "" {
			count, _ := strconv.Atoi(t)
			if err := db.SetCountTag(h.DB.Sql, count); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusAccepted, gin.H{"data": "Настройки сохранены"})
			return
		}
	}
}
