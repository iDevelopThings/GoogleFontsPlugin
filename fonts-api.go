package main

import (
	"bytes"
	b64 "encoding/base64"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"

	font_service "GoogleFontsPluginApi/font-service"
	"GoogleFontsPluginApi/logger"
)

type FontsApi struct {
	Group fiber.Router
}

func NewFontsApi(app *fiber.App, api fiber.Router) *FontsApi {
	inst := &FontsApi{
		Group: api.Group("/:provider/fonts"),
	}

	/*inst.Group.Use(cache.New(cache.Config{
		ExpirationGenerator: func(c fiber.Ctx, cfg *cache.Config) time.Duration {
			newCacheTime, _ := strconv.Atoi(c.GetRespHeader("Cache-Time", "600"))
			return time.Second * time.Duration(newCacheTime)
		},
		KeyGenerator: func(c fiber.Ctx) string {
			return utils.CopyString(c.Path())
		},
	}))*/

	inst.Group.Use(func(c fiber.Ctx) error {
		providerId := fiber.Params[string](c, "provider")
		fiber.Locals[string](c, "providerId", providerId)

		p := font_service.GetFontProvider(providerId)
		if p == nil {
			return fiber.ErrNotFound
		}

		fiber.Locals[font_service.IFontProvider](c, "provider", p)

		return c.Next()
	})

	inst.Group.Get("/all", inst.All)
	inst.Group.Get("/preview", inst.Preview)
	inst.Group.Get("/preview/multi", inst.PreviewMulti)
	inst.Group.Get("/license/:family", inst.License)

	return inst
}

func (a *FontsApi) All(c fiber.Ctx) error {

	startedAt := time.Now()
	defer func() { logger.Debug("[/Fonts/All]: %v", time.Since(startedAt)) }()

	opts := new(font_service.GetFontsFilters)
	if err := c.Bind().Query(opts); err != nil {
		return err
	}

	provider := font_service.GetFontProviderFromCtx(c)

	all, err := provider.GetFonts(opts)
	if err != nil {
		return err
	}

	return c.JSON(map[string]any{
		"items": all,
	})
}

func (a *FontsApi) Preview(c fiber.Ctx) error {
	provider := font_service.GetFontProviderFromCtx(c)

	r := new(font_service.CreateFontPreviewOptions)
	if err := c.Bind().Query(r); err != nil {
		return err
	}

	familyData, err := r.FirstFamilyAndVariant()
	if err != nil {
		return err
	}

	dctx, err := font_service.CreateFontPreview(provider, r, familyData)
	if err != nil {
		return err
	}

	// We need to encode as a png to a temporary buffer
	// so we can either output it as a base64 string or directly to the response
	var buf bytes.Buffer
	if err := dctx.EncodePNG(&buf); err != nil {
		return err
	}

	// If the result type is base64, we just send the base64 string
	// otherwise we set the content type to image/png and send the image
	if r.ResultType == font_service.FontPreviewResultTypeBase64 {
		base64Str := b64.StdEncoding.EncodeToString(buf.Bytes())
		c.Set("Content-Type", "text/plain")
		return c.SendString(base64Str)
	} else if r.ResultType == font_service.FontPreviewResultTypePng {
		c.Set("Content-Type", "image/png")
		return c.SendStream(&buf)
	}

	return fmt.Errorf("unknown result type")
}

func (a *FontsApi) PreviewMulti(c fiber.Ctx) error {
	provider := font_service.GetFontProviderFromCtx(c)

	r := new(font_service.CreateFontPreviewOptions)
	if err := c.Bind().Query(r); err != nil {
		return err
	}

	if r.ResultType == font_service.FontPreviewResultTypePng {
		return fmt.Errorf("result type png not supported for multi preview")
	}

	var wg sync.WaitGroup
	wg.Add(len(r.Families))

	// Family name -> base64 encoded image
	results := map[string]string{}
	familiesData := r.FamilyAndVariants()
	// Prepare the map
	for _, familyData := range familiesData {
		results[familyData.FullName()] = ""
	}
	for _, familyData := range familiesData {
		go func(familyData font_service.FontAndVariant) {
			defer wg.Done()

			dctx, err := font_service.CreateFontPreview(provider, r, familyData)
			if err != nil {
				fmt.Println("Error creating font preview:", err)
				return
			}

			// We need to encode as a png to a temporary buffer
			// so we can either output it as a base64 string or directly to the response
			var buf bytes.Buffer
			if err := dctx.EncodePNG(&buf); err != nil {
				fmt.Println("Error encoding PNG:", err)
				return
			}

			base64Str := b64.StdEncoding.EncodeToString(buf.Bytes())
			results[familyData.FullName()] = base64Str
		}(familyData)
	}

	wg.Wait()

	return c.JSON(results)
}

func (a *FontsApi) License(c fiber.Ctx) error {
	provider := font_service.GetFontProviderFromCtx(c)

	family := fiber.Params[string](c, "family")
	family = strings.Replace(family, "%20", " ", -1)

	font, found := provider.GetFontCache().Get(family)
	if !found {
		return fiber.ErrNotFound
	}

	if !font.HasLicense {
		return fiber.ErrNotFound
	}

	content, err := font.GetLicenseContent(provider)
	if err != nil {
		return err
	}

	return c.SendString(content)
}
