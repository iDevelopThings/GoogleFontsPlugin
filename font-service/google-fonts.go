package font_service

import (
	"context"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/schollz/progressbar/v3"
	"github.com/tingtt/iterutil"
	"github.com/wandb/parallel"

	"GoogleFontsPluginApi/cache"
	"GoogleFontsPluginApi/logger"
	"GoogleFontsPluginApi/utils"
)

// Structs and Types
type webFontApiErrorResponse struct {
	Error webFontApiError `json:"error"`
}

type webFontApiError struct {
	Code    *int                  `json:"code,omitempty"`
	Message *string               `json:"message,omitempty"`
	Errors  []webFontApiErrorData `json:"errors,omitempty"`
	Status  *string               `json:"status,omitempty"`
	Details []interface{}         `json:"details,omitempty"`
}

type webFontApiErrorData struct {
	Message *string `json:"message,omitempty"`
	Domain  *string `json:"domain,omitempty"`
	Reason  *string `json:"reason,omitempty"`
}

type webFontFamilyFilesMap map[string]string

type webFontFamilyFile struct {
	FontType string `json:"fontType"`
	URL      string `json:"url"`
}

type webFontFamily struct {
	Category     *string               `json:"category,omitempty"`
	Kind         string                `json:"kind"`
	Family       string                `json:"family"`
	Subsets      []string              `json:"subsets"`
	Variants     []string              `json:"variants"`
	Version      string                `json:"version"`
	LastModified string                `json:"lastModified"`
	Files        webFontFamilyFilesMap `json:"files"`
}

type webFontListOriginal struct {
	Kind  string          `json:"kind"`
	Items []webFontFamily `json:"items"`
}

type webFontCachedItem struct {
	webFontFamily
	FilesMapped webFontFamilyFilesMap `json:"filesMapped"`
	Files       []webFontFamilyFile   `json:"files"`
}

type GoogleFontsProvider struct {
	cache      *cache.TTLCache[string, FontFamilyData]
	categories []string
}

func (g *GoogleFontsProvider) GetCategories() []string { return g.categories }

func NewGoogleFontsProvider() IFontProvider {
	return &GoogleFontsProvider{
		cache:      cache.NewTTL[string, FontFamilyData](time.Hour * 24),
		categories: []string{},
	}
}

func (g *GoogleFontsProvider) GetId() string                                         { return "google" }
func (g *GoogleFontsProvider) GetDisplayName() string                                { return "Google Fonts" }
func (g *GoogleFontsProvider) GetFontCache() *cache.TTLCache[string, FontFamilyData] { return g.cache }

func (g *GoogleFontsProvider) GetFonts(opts *GetFontsFilters) ([]FontFamilyData, error) {
	filterIter := iterutil.FilterFunc(g.cache.Iterator(), func(data FontFamilyData) bool {
		if opts != nil {
			return opts.CanAddToResults(&data)
		}
		return true
	})

	items := slices.SortedFunc(filterIter, func(a, b FontFamilyData) int {
		return a.Order.Popularity - b.Order.Popularity
	})

	return items, nil
}
func (g *GoogleFontsProvider) CacheFonts() ([]FontFamilyData, error) {

	startedAt := time.Now()
	defer func() { logger.Debug("[Google.CacheFonts]: %v", time.Since(startedAt)) }()

	apiURL := "https://www.googleapis.com/webfonts/v1/webfonts?sort=popularity&key=" + os.Getenv("GOOGLE_API_KEY")
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch fonts: %w", err)
	}
	defer resp.Body.Close()

	bodyStr, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var jsonData webFontListOriginal
	if err := json.Unmarshal(bodyStr, &jsonData); err != nil {
		return nil, err
	}

	var uniqueCategories = make(map[string]bool)
	var shortItems []FontFamilyData
	for i, font := range jsonData.Items {
		category := "unknown"
		if font.Category != nil {
			category = *font.Category
		}

		uniqueCategories[category] = true

		item := FontFamilyData{
			Name:       font.Family,
			Category:   category,
			Variants:   []FontFamilyVariant{},
			HasLicense: true, // Set to true so it can be re-validated when we try to download the license
			Order: FontFamilyOrderValues{
				Popularity: i,
			},
		}

		createVariantPreviewObj := func(variantName string) VariantPreviewObject {
			variantStr := strings.ReplaceAll(font.Family, " ", "%20") + ":" + variantName
			text := strings.ReplaceAll(font.Family, " ", "%20")

			return VariantPreviewObject{
				Template: fmt.Sprintf("/api/%s/fonts/preview?families=%s&resultType=png&small={IsSmall}&text={Text}", g.GetId(), variantStr),
				Small:    fmt.Sprintf("/api/%s/fonts/preview?families=%s&resultType=png&small=true&text=%s", g.GetId(), variantStr, text),
				Large:    fmt.Sprintf("/api/%s/fonts/preview?families=%s&resultType=png&small=false&text=%s", g.GetId(), variantStr, text),
			}
		}

		hasRegular := false
		has500 := false
		for key, url := range font.Files {
			item.Variants = append(item.Variants, FontFamilyVariant{
				Name:        key,
				FullName:    font.Family + ":" + key,
				DownloadURL: url,
				Preview:     createVariantPreviewObj(key),
			})

			if key == "regular" {
				hasRegular = true
			}
			if key == "500" {
				has500 = true
			}
		}

		if !hasRegular && !has500 {
			fmt.Printf("Font %s does not have regular or 500 variant\n", font.Family)
			continue
		}

		if !hasRegular {
			item.Variants = append(item.Variants, FontFamilyVariant{
				Name:        "regular",
				FullName:    font.Family + ":regular",
				DownloadURL: font.Files["500"],
				Preview:     createVariantPreviewObj("500"),
			})
		}

		item.Variants = sortVariants(item.Variants)
		g.cache.Set(item.Name, item)

		shortItems = append(shortItems, item)
	}

	g.categories = slices.Collect(maps.Keys(uniqueCategories))

	return shortItems, nil
}
func (g *GoogleFontsProvider) GetFontAndVariant(family, variant string) (*FontFamilyAndVariantData, error) {
	data := new(FontFamilyAndVariantData)

	fontData, found := g.cache.Get(family)
	if !found {
		return nil, fmt.Errorf("font family not found")
	}

	data.Family = fontData

	for _, v := range fontData.Variants {
		if v.Name == variant {
			data.Variant = v
			return data, nil
		}
	}

	return nil, fmt.Errorf("variant not found")
}
func (g *GoogleFontsProvider) InitializeFromCache(data []FontFamilyData) {
	uniqueCategories := make(map[string]bool)
	for _, item := range data {
		g.cache.Set(item.Name, item)
		uniqueCategories[item.Category] = true
	}
	g.categories = slices.Collect(maps.Keys(uniqueCategories))
}
func getLicensePath(provider IFontProvider, fontName string) string {
	return GetProviderPath(provider.GetId(), "fonts", utils.GetPathSafeName(fontName), "license.txt")
}
func needsLicenseDownload(provider IFontProvider, font FontFamilyData) bool {
	return font.HasLicense && !utils.FileExists(getLicensePath(provider, font.Name))
}
func loadMissingLicenses(provider IFontProvider, fonts []FontFamilyData) error {
	// find fonts that need licenses
	var missingFonts []FontFamilyData
	for _, f := range fonts {
		if needsLicenseDownload(provider, f) {
			missingFonts = append(missingFonts, f)
		}
	}

	if len(missingFonts) == 0 {
		return nil
	}

	bar := progressbar.Default(int64(len(fonts)), "Downloading license files")
	ctx := context.Background()
	group := parallel.ErrGroup(parallel.Limited(ctx, 100))

	for _, f := range missingFonts {
		font := f

		group.Go(func(ctx context.Context) error {
			defer bar.Add(1)
			licensePath := getLicensePath(provider, font.Name)

			license, err := downloadGoogleFontLicense(font, nil)
			if err != nil {
				return err
			}

			provider.GetFontCache().Update(font.Name, func(f FontFamilyData) FontFamilyData {
				// f.License = license
				f.HasLicense = len(license) > 0
				return f
			})

			if err := utils.EnsurePathExists(licensePath); err != nil {
				return err
			}

			if err := os.WriteFile(licensePath, []byte(license), 0644); err != nil {
				return fmt.Errorf("failed to write license file to %s: %w", licensePath, err)
			}

			return nil
		})
	}

	return group.Wait()
}

func downloadGoogleFontLicense(font FontFamilyData, bar *progressbar.ProgressBar) (string, error) {
	if bar != nil {
		defer bar.Add(1)
	}

	fontName := utils.GetPathSafeName(font.Name)
	url := fmt.Sprintf("https://raw.githubusercontent.com/google/fonts/refs/heads/main/ofl/%s/OFL.txt", fontName)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to get license file for %s: %w", font.Name, err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", nil
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("non-200 response from server: %v", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(data), nil
}
