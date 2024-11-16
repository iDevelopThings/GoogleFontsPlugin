package font_service

import (
	"io"
	"os"
	"slices"
	"strings"
)

type FontFamilyOrderValues struct {
	Popularity int `json:"popularity"`
}

type FontFamilyData struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	// Set to true so it can be re-validated when we try to download the license
	HasLicense bool                `json:"hasLicense"`
	Variants   []FontFamilyVariant `json:"variants"`

	Order FontFamilyOrderValues `json:"order"`
}

func (d FontFamilyData) GetLicenseContent(provider IFontProvider) (string, error) {
	p := getLicensePath(provider, d.Name)

	file, err := os.Open(p)
	if err != nil {
		return "", err
	}

	defer file.Close()

	license, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	return string(license), nil
}

type FontFamilyVariant struct {
	Name        string               `json:"name"`
	FullName    string               `json:"fullName"`
	DownloadURL string               `json:"downloadUrl"`
	Preview     VariantPreviewObject `json:"preview"`
}

type VariantPreviewObject struct {
	Template string `json:"template"`
	Small    string `json:"small"`
	Large    string `json:"large"`
}

type FontFamilyAndVariantData struct {
	Family  FontFamilyData
	Variant FontFamilyVariant
}

func (f *FontFamilyAndVariantData) FontCacheKey() string {
	return f.Family.Name + ":" + f.Variant.Name
}

type GetFontsFilters struct {
	Categories []string `json:"categories,omitempty" query:"categories"`
	Search     *string  `json:"search,omitempty" query:"search,default:nil"`
}

func (f *GetFontsFilters) HasCategory() bool { return len(f.Categories) > 0 }
func (f *GetFontsFilters) HasSearch() bool   { return f.Search != nil && *f.Search != "" }

func (f *GetFontsFilters) CanAddToResults(font *FontFamilyData) bool {
	if f.HasCategory() && !slices.Contains(f.Categories, font.Category) {
		return false
	}
	if f.HasSearch() && !strings.Contains(strings.ToLower(font.Name), strings.ToLower(*f.Search)) {
		return false
	}
	return true
}
