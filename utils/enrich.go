package utils

import (
	"strconv"

	"github.com/Smiduweorc/monarchButterflyIcloud/types"
)

// EnrichImagesWithURLs returns the album's images with every derivative's URL
// populated from the checksum->URL map produced by core.GetUrls. Derivatives
// whose checksum is absent from urls are dropped, and the surviving derivatives
// are re-keyed by their pixel height (with "-N" suffixes for collisions).
func EnrichImagesWithURLs(
	apiResponse types.APIResponse,
	urls map[string]string,
) []types.Image {

	var result []types.Image

	for _, photo := range apiResponse.Photos {

		newDerivatives := make(map[string]types.ImageInfo)
		duplicateCount := make(map[string]int)

		for _, d := range photo.Derivatives {

			url, ok := urls[d.Checksum]
			if !ok {
				continue
			}

			key := strconv.Itoa(d.Height)

			// handle duplicate heights
			if _, exists := newDerivatives[key]; exists {
				duplicateCount[key]++
				key = key + "-" + strconv.Itoa(duplicateCount[key])
			} else {
				duplicateCount[key] = 0
			}

			// attach URL
			d.URL = url

			newDerivatives[key] = d
		}

		photo.Derivatives = newDerivatives
		result = append(result, photo)
	}

	return result
}
