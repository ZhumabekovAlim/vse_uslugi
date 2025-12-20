package repositories

import (
	"database/sql"
	"encoding/json"
	"strings"
)

type imagePayload struct {
	Path string `json:"path"`
}

func extractFirstImagePath(imagesJSON sql.NullString) (*string, error) {
	if !imagesJSON.Valid {
		return nil, nil
	}

	data := strings.TrimSpace(imagesJSON.String)
	if data == "" {
		return nil, nil
	}

	var images []imagePayload
	if err := json.Unmarshal([]byte(data), &images); err != nil {
		return nil, err
	}

	for _, img := range images {
		path := strings.TrimSpace(img.Path)
		if path != "" {
			return &img.Path, nil
		}
	}

	return nil, nil
}
