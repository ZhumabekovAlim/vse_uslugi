package handlers

import (
	"encoding/json"
	"mime/multipart"
	"strconv"
	"strings"

	"naimuBack/internal/models"
)

// imagePayload объединяет все типы изображений, используемых в сущностях.
type imagePayload interface {
	models.Image | models.ImageAd | models.ImageRent | models.ImageRentAd | models.ImageWork | models.ImageWorkAd | models.Video
}

// collectImageFiles собирает все файлы по указанным ключам формы.
func collectImageFiles(form *multipart.Form, keys ...string) []*multipart.FileHeader {
	if form == nil {
		return nil
	}

	var result []*multipart.FileHeader
	for _, key := range keys {
		if headers, ok := form.File[key]; ok {
			result = append(result, headers...)
		}
	}
	return result
}

// gatherImagesFromForm считывает строковые значения из multipart-формы и преобразует их в нужный тип изображений.
// Возвращает срез изображений, признак того, что данные присутствовали, и возможную ошибку.
func gatherImagesFromForm[T imagePayload](form *multipart.Form, keys ...string) ([]T, bool, error) {
	if form == nil {
		return nil, false, nil
	}

	var rawValues []string
	for _, key := range keys {
		if values, ok := form.Value[key]; ok {
			rawValues = append(rawValues, values...)
		}
	}
	if len(rawValues) == 0 {
		return nil, false, nil
	}

	images, err := parseImagesFromValues[T](rawValues)
	if err != nil {
		return nil, false, err
	}

	if len(images) == 0 {
		return nil, false, nil
	}

	return images, true, nil
}

func parseImagesFromValues[T imagePayload](values []string) ([]T, error) {
	var result []T

	for _, raw := range values {
		raw = strings.TrimSpace(raw)
		if raw == "" || raw == "null" || raw == "undefined" {
			continue
		}

		if strings.HasPrefix(raw, "[") {
			var arr []T
			if err := json.Unmarshal([]byte(raw), &arr); err == nil {
				for i := range arr {
					normalizeImage(&arr[i])
				}
				result = append(result, arr...)
				continue
			}

			var links []string
			if err := json.Unmarshal([]byte(raw), &links); err == nil {
				for _, link := range links {
					link = strings.TrimSpace(link)
					if link == "" {
						continue
					}
					result = append(result, newLinkImage[T](link))
				}
				continue
			}

			continue
		}

		if strings.HasPrefix(raw, "{") {
			var item T
			if err := json.Unmarshal([]byte(raw), &item); err == nil {
				normalizeImage(&item)
				result = append(result, item)
				continue
			}

			continue
		}

		if strings.HasPrefix(raw, "\"") && strings.HasSuffix(raw, "\"") {
			if unquoted, err := strconv.Unquote(raw); err == nil {
				raw = unquoted
			}
		}

		img := newLinkImage[T](raw)
		result = append(result, img)
	}

	return result, nil
}

func normalizeImage[T imagePayload](img *T) {
	switch v := any(img).(type) {
	case *models.Image:
		if v.Name == "" {
			v.Name = v.Path
		}
	case *models.ImageAd:
		if v.Name == "" {
			v.Name = v.Path
		}
	case *models.ImageRent:
		if v.Name == "" {
			v.Name = v.Path
		}
	case *models.ImageRentAd:
		if v.Name == "" {
			v.Name = v.Path
		}
	case *models.ImageWork:
		if v.Name == "" {
			v.Name = v.Path
		}
	case *models.ImageWorkAd:
		if v.Name == "" {
			v.Name = v.Path
		}
	case *models.Video:
		if v.Name == "" {
			v.Name = v.Path
		}
	}
}

func newLinkImage[T imagePayload](path string) T {
	var img T

	switch v := any(&img).(type) {
	case *models.Image:
		v.Name = path
		v.Path = path
		v.Type = "link"
	case *models.ImageAd:
		v.Name = path
		v.Path = path
		v.Type = "link"
	case *models.ImageRent:
		v.Name = path
		v.Path = path
		v.Type = "link"
	case *models.ImageRentAd:
		v.Name = path
		v.Path = path
		v.Type = "link"
	case *models.ImageWork:
		v.Name = path
		v.Path = path
		v.Type = "link"
	case *models.ImageWorkAd:
		v.Name = path
		v.Path = path
		v.Type = "link"
	case *models.Video:
		v.Name = path
		v.Path = path
		v.Type = "link"
	}

	return img
}
