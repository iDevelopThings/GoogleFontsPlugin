package font_service

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"

	"GoogleFontsPluginApi/logger"
)

type FontPreviewResultType string

const (
	FontPreviewResultTypeBase64 FontPreviewResultType = "base64"
	FontPreviewResultTypePng    FontPreviewResultType = "png"
)

type CreateFontPreviewOptions struct {
	Families   []string              `query:"families"`
	Text       string                `query:"text"`
	ResultType FontPreviewResultType `query:"resultType,default:png"`
	Small      bool                  `query:"small,default:false"`
}
type FontAndVariant struct {
	Family  string
	Variant string
}

func (f FontAndVariant) FullName() string { return f.Family + ":" + f.Variant }

func ExtractFamilyAndVariant(familyStr string) FontAndVariant {
	data := FontAndVariant{}

	parts := strings.Split(familyStr, ":")
	data.Family = parts[0]
	if len(parts) > 1 {
		data.Variant = parts[1]
	}

	return data
}
func ExtractFamilyAndVariants(familyStrs []string) []FontAndVariant {
	data := make([]FontAndVariant, len(familyStrs))

	for i, familyStr := range familyStrs {
		data[i] = ExtractFamilyAndVariant(familyStr)
	}

	return data
}

func (o *CreateFontPreviewOptions) FirstFamilyAndVariant() (FontAndVariant, error) {
	if len(o.Families) == 0 {
		return FontAndVariant{}, fmt.Errorf("no families specified")
	}

	return ExtractFamilyAndVariant(o.Families[0]), nil
}
func (o *CreateFontPreviewOptions) FamilyAndVariants() []FontAndVariant {
	return ExtractFamilyAndVariants(o.Families)
}

type previewSize struct {
	width    float64
	height   float64
	fontSize float64
}

var previewSizes = map[bool]previewSize{
	// true = small - small is more like a banner
	true: {800, 100, 30},
	// false = large - large is more like a cover image
	false: {400, 200, 40},
}

func CreateFontPreview(
	provider IFontProvider,
	r *CreateFontPreviewOptions,
	familyData FontAndVariant,
) (*gg.Context, error) {
	data, err := provider.GetFontAndVariant(familyData.Family, familyData.Variant)
	if err != nil {
		return nil, err
	}

	ft, err := GetOrCacheFont(data)
	if err != nil {
		logger.Error("Failed to get font: %v", err)
		return nil, err
	}

	size := previewSizes[r.Small]

	fontFace := truetype.NewFace(ft, &truetype.Options{
		Size:    size.fontSize,
		DPI:     96,
		Hinting: font.HintingFull,
	})

	dc := gg.NewContext(int(size.width), int(size.height))
	dc.SetColor(color.Transparent)
	dc.Clear()

	dc.SetFontFace(fontFace)
	dc.SetColor(color.White)
	// dc.DrawStringAnchored(r.Text, size.width/2, size.height/2, 0.5, 0.5)
	dc.DrawStringWrapped(r.Text, size.width/2, size.height/2, 0.5, 0.5, size.width-20, 1.5, gg.AlignCenter)

	return dc, nil
}
