package http

import (
	"net/http"
	"skeji/pkg/config"
	apperrors "skeji/pkg/errors"
	"strconv"
)

func ExtractLimitOffset(r *http.Request) (int, int, error) {
	query := r.URL.Query()

	limit := 0
	if s := query.Get("limit"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil {
			return 0, 0, apperrors.InvalidInput("invalid limit parameter: " + s)
		}
		limit = v
	}

	offset := 0
	if s := query.Get("offset"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil {
			return 0, 0, apperrors.InvalidInput("invalid offset parameter: " + s)
		}
		offset = v
	}

	limit = config.NormalizePaginationLimit(limit)
	offset = config.NormalizeOffset(offset)

	return limit, offset, nil
}
