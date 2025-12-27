package middleware

import (
	"net/url"
	"testing"

	"github.com/tschuyebuhl/httpkit/data"
)

func TestParseQueryParamsFilters(t *testing.T) {
	values := url.Values{}
	values.Add("filter", "habit_code_anywhere= gym ")
	values.Add("filter", "name_eq=Focus")

	params := parseQueryParams(values)
	if params == nil {
		t.Fatalf("expected params, got nil")
	}

	if len(params.Conditions) != 2 {
		t.Fatalf("expected 2 filter conditions, got %d", len(params.Conditions))
	}

	first := params.Conditions[0]
	if first.Column != "habit_code" || first.Mode != data.Anywhere || first.Value != "gym" {
		t.Fatalf("unexpected first condition: %+v", first)
	}

	second := params.Conditions[1]
	if second.Column != "name" || second.Mode != data.Exact || second.Value != "Focus" {
		t.Fatalf("unexpected second condition: %+v", second)
	}
}

func TestParseQueryParamsSort(t *testing.T) {
	values := url.Values{}
	values.Set("sort", "-completed_at")

	params := parseQueryParams(values)
	if params == nil {
		t.Fatalf("expected params, got nil")
	}

	if params.Column != "completed_at" || params.Direction != "desc" {
		t.Fatalf("expected sort completed_at desc, got %+v", params.Sort)
	}
}

func TestParseQueryParamsPagination(t *testing.T) {
	values := url.Values{}
	values.Set("per_page", "15")
	values.Set("page", "3")

	params := parseQueryParams(values)
	if params == nil {
		t.Fatalf("expected params, got nil")
	}

	limit, ok := params.Limit.(int64)
	if !ok {
		t.Fatalf("expected numeric limit, got %T", params.Limit)
	}

	if limit != 15 {
		t.Fatalf("expected limit 15, got %d", limit)
	}

	if params.Offset != 30 {
		t.Fatalf("expected offset 30, got %d", params.Offset)
	}
}

func TestParseQueryParamsLimitAll(t *testing.T) {
	values := url.Values{}
	values.Set("limit", "ALL")

	params := parseQueryParams(values)
	if params == nil {
		t.Fatalf("expected params, got nil")
	}

	limit, ok := params.Limit.(string)
	if !ok {
		t.Fatalf("expected string limit, got %T", params.Limit)
	}

	if limit != "ALL" {
		t.Fatalf("expected limit ALL, got %s", limit)
	}
}
