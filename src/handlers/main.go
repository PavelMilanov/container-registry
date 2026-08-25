// Package handlers реализует REST API приложения.
// Интеграция спецификации https://distribution.github.io/distribution/spec/api/
// с кастомным API.
package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/PavelMilanov/container-registry/config"
	"github.com/PavelMilanov/container-registry/db"
	registryauth "github.com/PavelMilanov/container-registry/internal/auth"
	"github.com/PavelMilanov/container-registry/middleware"
	"github.com/PavelMilanov/container-registry/storage"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Handler основная сущность взаимодействия с API.
type Handler struct {
	BLOBS        storage.BlobStore
	UPLOADS      storage.BlobUploadStore
	MANIFESTS    storage.ManifestStore
	NAMESPACES   storage.NamespaceStore
	CLOUDS       storage.CloudStore
	REPOSITORIES storage.RepositoryStore
	TAGS         storage.TagStore

	GARBAGE_COLLECTOR storage.GarbageCollector
	TAG_PRUNER        storage.TagPruner

	AUTH Authenticator

	DB  *db.SQLite
	ENV *config.Env
}

// StorageDependencies содержит storage-возможности HTTP-обработчиков.
type StorageDependencies struct {
	Blobs        storage.BlobStore
	Uploads      storage.BlobUploadStore
	Manifests    storage.ManifestStore
	Namespaces   storage.NamespaceStore
	Clouds       storage.CloudStore
	Repositories storage.RepositoryStore
	Tags         storage.TagStore

	GarbageCollector storage.GarbageCollector
	TagPruner        storage.TagPruner
}

// Authenticator предоставляет обработчикам операции аутентификации.
type Authenticator interface {
	Register(ctx context.Context, username, password string) error
	Login(
		ctx context.Context,
		username string,
		password string,
		access []registryauth.ResourceAction,
	) (registryauth.IssuedToken, error)
	ValidateToken(rawToken string) (registryauth.Claims, error)
}

/*
NewHandler инициализирует обработчик API.

	stores - отдельные storage-возможности обработчиков.
	authService - сервис регистрации, входа и проверки JWT.
	db - подключение к базе данных.
	env - конфигурация приложения.
*/
func NewHandler(
	stores StorageDependencies,
	authService Authenticator,
	db *db.SQLite,
	env *config.Env,
) *Handler {
	return &Handler{
		BLOBS:             stores.Blobs,
		UPLOADS:           stores.Uploads,
		MANIFESTS:         stores.Manifests,
		NAMESPACES:        stores.Namespaces,
		CLOUDS:            stores.Clouds,
		REPOSITORIES:      stores.Repositories,
		TAGS:              stores.Tags,
		GARBAGE_COLLECTOR: stores.GarbageCollector,
		TAG_PRUNER:        stores.TagPruner,
		AUTH:              authService,
		DB:                db,
		ENV:               env,
	}
}

func (h *Handler) InitRouters() *gin.Engine {
	router := gin.New()
	validateToken := middleware.TokenValidator(func(
		token string,
	) (middleware.Identity, error) {
		claims, err := h.AUTH.ValidateToken(token)
		if err != nil {
			return middleware.Identity{}, err
		}
		access := make([]middleware.ResourceAccess, 0, len(claims.Access))
		for _, resource := range claims.Access {
			access = append(access, middleware.ResourceAccess{
				Type:    resource.Type,
				Name:    resource.Name,
				Actions: append([]string(nil), resource.Actions...),
			})
		}
		return middleware.Identity{
			Subject: claims.Subject,
			Access:  access,
		}, nil
	})
	router.Use(
		middleware.RequestLogger(logrus.StandardLogger()),
		gin.Recovery(),
	)
	// router.Use(cors.New(cors.Config{
	// 	AllowOrigins:     []string{"http://localhost:5050"},
	// 	AllowMethods:     []string{"GET", "POST", "DELETE"},
	// 	AllowHeaders:     []string{"Origin", "Authorization", "Content-Type"},
	// 	ExposeHeaders:    []string{"Content-Length"},
	// 	AllowCredentials: true,
	// 	MaxAge:           24 * time.Hour,
	// }))
	router.POST("/login", h.login)
	router.GET("/check", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	router.POST("/registration", h.registration)
	router.GET("/v2/auth", h.authHandler)
	router.POST("/v2/auth", h.authHandler)

	v2 := router.Group(
		"/v2/",
		middleware.RequireRegistryAuth(
			h.ENV.Server.Realm,
			h.ENV.Server.Service,
			validateToken,
		),
	)
	{
		// Пинг для проверки
		v2.GET("/", func(c *gin.Context) {
			c.Header("Docker-Distribution-Api-Version", "registry/2.0")
			c.JSON(http.StatusOK, gin.H{"message": "Docker Registry API"})
		})

		repository := v2.Group(
			"/:repository/:name",
			middleware.RequireNamespace(h.NAMESPACES),
		)
		// manifests
		repository.HEAD("/manifests/:reference", h.getManifest)
		repository.GET("/manifests/:reference", h.getManifest)
		repository.PUT("/manifests/:reference", h.uploadManifest)
		// blobs
		repository.GET("/blobs/:uuid", h.getBlob)
		repository.HEAD("/blobs/:uuid", h.checkBlob)
		repository.POST("/blobs/uploads/", h.startBlobUpload)
		repository.PATCH("/blobs/uploads/:uuid", h.uploadBlobPart)
		repository.PUT("/blobs/uploads/:uuid", h.finalizeBlobUpload)
		repository.DELETE("/blobs/uploads/:uuid", h.abortBlobUpload)
	}

	api := router.Group("/api/", middleware.RequireAPIAuth(validateToken))
	{
		cloud := api.Group("/cloud/")
		{
			cloud.GET("/:cloud/:repository", h.getImagesList)
			cloud.GET("/:cloud", h.getRepoList)
			cloud.GET("/list", h.getCloudList)
			cloud.POST("/create", h.addCloud)
			cloud.DELETE("/delete", h.deleteCloud)
			cloud.DELETE("/:cloud/:repository", h.deleteRepositoryOrImage)
		}
		garbage := api.Group("/garbage/")
		{
			garbage.POST("/collection", h.garbageCollection)
			garbage.POST("/tags", h.deleteOlderTags)

		}
		api.GET("/settings", h.settings)
		api.POST("/settings", h.settings)
	}
	router.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/v2/") {
			addRequestErrorMessage(c, "route not found")
			c.Status(http.StatusNotFound)
			return
		}
		c.String(http.StatusOK, "is OK.")
	})
	return router
}
