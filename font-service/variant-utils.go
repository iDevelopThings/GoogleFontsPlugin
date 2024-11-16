package font_service

import (
	"fmt"
	"regexp"
	"sort"
)

func sortVariants(variants []FontFamilyVariant) []FontFamilyVariant {
	var italicVariants, regularVariants, nonRegularVariants []FontFamilyVariant

	for _, variant := range variants {
		if regexp.MustCompile(`italic`).MatchString(variant.Name) {
			if variant.Name == "italic" {
				italicVariants = append([]FontFamilyVariant{variant}, italicVariants...)
			} else {
				italicVariants = append(italicVariants, variant)
			}
		} else if variant.Name == "regular" {
			regularVariants = append(regularVariants, variant)
		} else {
			nonRegularVariants = append(nonRegularVariants, variant)
		}
	}

	sort.Slice(italicVariants, func(i, j int) bool {
		aNum, bNum := extractNumFromVariant(italicVariants[i].Name), extractNumFromVariant(italicVariants[j].Name)
		return aNum < bNum
	})

	return append(append(regularVariants, nonRegularVariants...), italicVariants...)
}

func extractNumFromVariant(name string) int {
	var num int
	_, err := fmt.Sscanf(name, "%ditalic", &num)
	if err != nil {
		return 0
	}
	return num
}
