package font_service

import (
	"io"
	"net/http"
	"path"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang/freetype/truetype"

	"GoogleFontsPluginApi/cache"
	"GoogleFontsPluginApi/logger"
)

var FontProviders *FontProviderService

type FontProviderService struct {
	sync.Mutex

	Providers map[string]*FontProvider
	FontCache *cache.TTLCache[string, *truetype.Font]
}

type FontProvider struct {
	Id          string `json:"id"`
	DisplayName string `json:"displayName"`
	EndPoint    string `json:"endpoint"`

	internalProvider IFontProvider
}

func (f *FontProvider) GetPath(subPaths ...string) string { return GetProviderPath(f.Id, subPaths...) }

func (f *FontProvider) GetId() string          { return f.Id }
func (f *FontProvider) GetDisplayName() string { return f.DisplayName }
func (f *FontProvider) GetEndpoint() string    { return f.EndPoint }
func (f *FontProvider) GetFonts(opts *GetFontsFilters) ([]FontFamilyData, error) {
	return f.internalProvider.GetFonts(opts)
}
func (f *FontProvider) CacheFonts() ([]FontFamilyData, error) { return f.internalProvider.CacheFonts() }
func (f *FontProvider) GetFontCache() *cache.TTLCache[string, FontFamilyData] {
	return f.internalProvider.GetFontCache()
}
func (f *FontProvider) GetFontAndVariant(family, variant string) (*FontFamilyAndVariantData, error) {
	return f.internalProvider.GetFontAndVariant(family, variant)
}

func (f *FontProvider) GetCategories() []string { return f.internalProvider.GetCategories() }
func (f *FontProvider) InitializeFromCache(data []FontFamilyData) {
	f.internalProvider.InitializeFromCache(data)
}

type IFontProvider interface {
	GetId() string
	GetDisplayName() string
	GetFonts(opts *GetFontsFilters) ([]FontFamilyData, error)
	CacheFonts() ([]FontFamilyData, error)
	GetFontCache() *cache.TTLCache[string, FontFamilyData]
	GetFontAndVariant(family, variant string) (*FontFamilyAndVariantData, error)
	GetCategories() []string
	InitializeFromCache(data []FontFamilyData)
}

func init() {
	FontProviders = &FontProviderService{
		Providers: map[string]*FontProvider{},
		FontCache: cache.NewTTL[string, *truetype.Font](time.Hour * 24),
	}

	FontProviders.AddProvider(NewGoogleFontsProvider())

	for _, provider := range FontProviders.Providers {
		if err := loadProviderCacheFromDisk(provider); err != nil {
			logger.Error("Failed to load cache for provider %s: %v", provider.GetId(), err)
			continue
		}

		if err := saveProviderCacheToDisk(provider); err != nil {
			logger.Error("Failed to save cache for provider %s: %v", provider.GetId(), err)
		}
	}

	logger.Info("Font providers initialized")
}

func GetFontProvidersArray() []IFontProvider {
	providers := make([]IFontProvider, 0, len(FontProviders.Providers))
	for _, provider := range FontProviders.Providers {
		providers = append(providers, provider)
	}
	return providers
}
func GetProviderPath(id string, subPaths ...string) string {
	return path.Join("data", id, path.Join(subPaths...))
}
func GetFontProvider(id string) IFontProvider { return FontProviders.GetProviderById(id) }
func GetFontProviderFromCtx(c fiber.Ctx) IFontProvider {
	return fiber.Locals[IFontProvider](c, "provider")
}

func GetOrCacheFont(data *FontFamilyAndVariantData) (*truetype.Font, error) {
	FontProviders.Lock()
	defer FontProviders.Unlock()

	font, found := FontProviders.FontCache.Get(data.FontCacheKey())
	if found {
		return font, nil
	}

	resp, err := http.Get(data.Variant.DownloadURL)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fontData, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	// Parse the font and create a font face
	ft, err := truetype.Parse(fontData)
	if err != nil {
		return nil, err
	}

	FontProviders.FontCache.Set(data.FontCacheKey(), ft)

	return ft, nil
}

func (s *FontProviderService) GetProviderById(id string) IFontProvider {
	if provider, ok := s.Providers[id]; ok {
		return provider
	}
	return nil
}

func (s *FontProviderService) AddProvider(provider IFontProvider) {
	p := &FontProvider{
		Id:               provider.GetId(),
		DisplayName:      provider.GetDisplayName(),
		EndPoint:         "/api/" + provider.GetId() + "/fonts",
		internalProvider: provider,
	}
	s.Providers[provider.GetId()] = p
}
