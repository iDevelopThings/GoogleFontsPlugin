package main

import (
	"log"
	"os"

	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v3"
	gofiberlogger "github.com/gofiber/fiber/v3/middleware/logger"
	recover2 "github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	_ "github.com/joho/godotenv/autoload"

	"GoogleFontsPluginApi/conf"
	fontservice "GoogleFontsPluginApi/font-service"
	"GoogleFontsPluginApi/logger"
)

var fontsApi *FontsApi

func main() {
	if err := conf.Config.Load("conf/app.json"); err != nil {
		panic(err)
	}
	logger.Init(conf.Config)

	app := fiber.New(fiber.Config{
		JSONEncoder: json.Marshal,
		JSONDecoder: json.Unmarshal,
	})
	app.Use(recover2.New(recover2.Config{
		EnableStackTrace: true,
	}))
	app.Use(requestid.New())
	app.Use(gofiberlogger.New(gofiberlogger.Config{
		Format: "${pid} ${locals:requestid} ${status} - ${method} ${path}\n",
	}))

	api := app.Group("/api")

	api.Get("/providers", func(c fiber.Ctx) error {
		fontservice.FontProviders.Lock()
		defer fontservice.FontProviders.Unlock()

		var data []map[string]any
		for _, provider := range fontservice.FontProviders.Providers {
			data = append(data, map[string]any{
				"id":          provider.GetId(),
				"displayName": provider.GetDisplayName(),
				"endpoint":    provider.GetEndpoint(),
				"categories":  provider.GetCategories(),
			})
		}
		return c.JSON(map[string]interface{}{
			"items": data,
		})
	})

	fontsApi = NewFontsApi(app, api)

	logger.Debug("Starting server on %s", os.Getenv("HOST"))

	log.Fatal(app.Listen(os.Getenv("HOST")))
}
