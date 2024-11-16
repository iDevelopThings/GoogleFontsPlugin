package font_service

import (
	"io"
	"os"

	"github.com/goccy/go-json"

	"GoogleFontsPluginApi/utils"
)

func saveProviderCacheToDisk(provider *FontProvider) error {
	var err error
	var cachePath = provider.GetPath("cache.json")

	data := provider.GetFontCache().All()

	err = saveCacheData(cachePath, data)
	if err != nil {
		return err
	}

	return nil
}

func loadProviderCacheFromDisk(provider IFontProvider) error {
	var err error
	var cachePath = GetProviderPath(provider.GetId(), "cache.json")

	var cachedData = make([]FontFamilyData, 0)

	loaded, err := loadCacheData(cachePath, &cachedData)
	if err != nil {
		return err
	}

	if !loaded {
		d, err := provider.CacheFonts()
		if err != nil {
			return err
		}

		cachedData = d
	} else {
		provider.InitializeFromCache(cachedData)
	}

	return loadMissingLicenses(provider, cachedData)
}

func saveCacheData(path string, data interface{}) error {
	err := utils.EnsurePathExists(path)
	if err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}

	defer file.Close()

	body, err := json.MarshalWithOption(data, json.DisableHTMLEscape())
	if err != nil {
		return err
	}

	_, err = file.Write(body)
	if err != nil {
		return err
	}

	return nil
}

func loadCacheData(path string, data interface{}) (bool, error) {
	if !utils.FileExists(path) {
		return false, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return false, err
	}

	defer file.Close()

	bodyBytes, err := io.ReadAll(file)
	if err != nil {
		return false, err
	}

	if len(bodyBytes) == 0 {
		return false, nil
	}

	if err := json.Unmarshal(bodyBytes, data); err != nil {
		return false, err
	}

	return true, nil
}
