package handlers

import (
	"encoding/json"
	"fmt"
	"mime/multipart"
	"strings"

	"naimuBack/internal/models"
)

// imagePayload объединяет все типы изображений, используемых в сущностях.
type imagePayload interface {
	models.Image | models.ImageAd | models.ImageRent | models.ImageRentAd | models.ImageWork | models.ImageWorkAd
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
			if err := json.Unmarshal([]byte(raw), &arr); err != nil {
				return nil, fmt.Errorf("failed to decode image array: %w", err)
			}
			for i := range arr {
				normalizeImage(&arr[i])
			}
			result = append(result, arr...)
			continue
		}

		if strings.HasPrefix(raw, "{") {
			var item T
			if err := json.Unmarshal([]byte(raw), &item); err != nil {
				return nil, fmt.Errorf("failed to decode image object: %w", err)
			}
			normalizeImage(&item)
			result = append(result, item)
			continue
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
	}

	return img
}
