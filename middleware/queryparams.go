package middleware

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/tschuyebuhl/httpkit/data"
)

type queryParamsKey struct{}

func QueryParams(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		params := parseQueryParams(r.URL.Query())
		if params != nil {
			ctx := context.WithValue(r.Context(), queryParamsKey{}, params)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func QueryParamsFromContext(ctx context.Context) *data.QueryParams {
	if ctx == nil {
		return nil
	}
	if params, ok := ctx.Value(queryParamsKey{}).(*data.QueryParams); ok {
		return params
	}
	return nil
}

func parseQueryParams(values url.Values) *data.QueryParams {
	if len(values) == 0 {
		return nil
	}

	params := &data.QueryParams{
		Pagination: data.Pagination{
			Limit:  "ALL",
			Offset: 0,
		},
	}

	var hasParams bool

	// Filters
	if filters := values["filter"]; len(filters) > 0 {
		conds := make([]data.FilterCondition, 0, len(filters))
		for _, f := range filters {
			if cond, ok := parseFilterCondition(f); ok {
				conds = append(conds, cond)
			}
		}
		if len(conds) > 0 {
			params.Filter = data.Filter{Conditions: conds}
			hasParams = true
		}
	}

	// Sorting
	if column, direction := parseSort(values.Get("sort")); column != "" {
		params.Sort = data.Sort{Column: column, Direction: direction}
		hasParams = true
	}

	// Pagination
	var (
		numericLimit  int64
		limitIsNumber bool
		limitParam    = values.Get("limit")
	)

	if limitValue, isNumber := parseLimit(limitParam); limitValue != nil {
		params.Limit = limitValue
		hasParams = true
		limitIsNumber = isNumber
		if isNumber {
			numericLimit = limitValue.(int64)
		}
	}

	if offset, ok := parseInt(values.Get("offset")); ok {
		params.Offset = offset
		hasParams = true
	} else if page, ok := parseInt(values.Get("page")); ok && page > 0 && limitIsNumber {
		params.Offset = (page - 1) * numericLimit
		hasParams = true
	}

	if hasParams {
		return params
	}
	return nil
}

func parseFilterCondition(raw string) (data.FilterCondition, bool) {
	if raw == "" {
		return data.FilterCondition{}, false
	}
	parts := strings.SplitN(raw, "=", 2)
	if len(parts) != 2 {
		return data.FilterCondition{}, false
	}
	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	if key == "" {
		return data.FilterCondition{}, false
	}

	column, mode := splitColumnAndMode(key)
	if column == "" {
		return data.FilterCondition{}, false
	}
	return data.FilterCondition{
		Column: column,
		Mode:   mode,
		Value:  value,
	}, true
}

func splitColumnAndMode(key string) (string, data.MatchMode) {
	lastUnderscore := strings.LastIndex(key, "_")
	if lastUnderscore == -1 {
		return strings.TrimSpace(key), data.Exact
	}
	column := strings.TrimSpace(key[:lastUnderscore])
	mode := key[lastUnderscore+1:]
	if column == "" {
		return "", data.Exact
	}
	return column, parseMatchMode(mode)
}

func parseMatchMode(mode string) data.MatchMode {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "exact", "eq":
		return data.Exact
	case "ci", "caseinsensitive", "case_insensitive", "ilike":
		return data.CaseInsensitive
	case "start", "prefix", "starts_with":
		return data.Start
	case "end", "suffix", "ends_with":
		return data.End
	case "any", "anywhere", "contains":
		return data.Anywhere
	default:
		return data.Exact
	}
}

func parseSort(raw string) (string, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ""
	}

	direction := "asc"
	if strings.HasPrefix(raw, "-") {
		direction = "desc"
		raw = strings.TrimPrefix(raw, "-")
	} else if strings.HasPrefix(raw, "+") {
		raw = strings.TrimPrefix(raw, "+")
	}

	parts := strings.SplitN(raw, ":", 2)
	if parts == nil {
		return "", ""
	}
	column := strings.TrimSpace(parts[0])
	if column == "" {
		return "", ""
	}

	if len(parts) == 2 {
		dir := strings.ToLower(strings.TrimSpace(parts[1]))
		if dir == "asc" || dir == "desc" {
			direction = dir
		}
	}

	return column, direction
}

func parseLimit(value string) (any, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, false
	}
	if strings.EqualFold(value, "all") {
		return "ALL", false
	}
	if limit, err := strconv.ParseInt(value, 10, 64); err == nil && limit >= 0 {
		return limit, true
	}
	return nil, false
}

func parseInt(value string) (int64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	v, err := strconv.ParseInt(value, 10, 64)
	if err != nil || v < 0 {
		return 0, false
	}
	return v, true
}
