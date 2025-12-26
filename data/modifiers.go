package data

import (
	"fmt"

	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

type Pagination struct {
	Limit  any // number or "ALL"
	Offset int64
}

func Slice[T any](data []T, count int64) SliceResult[T] {
	return SliceResult[T]{
		Data:  data,
		Total: count,
	}
}

type SliceResult[T any] struct {
	Data  []T   `json:"data"`
	Total int64 `json:"total"`
}

type Sort struct {
	Column    string
	Direction string
}

type FilterCondition struct {
	Column string
	Mode   MatchMode
	Value  string
}

type Filter struct {
	Conditions []FilterCondition
}

type QueryParams struct {
	Pagination
	Filter
	Sort
}

type MatchMode int

const (
	Exact           MatchMode = iota // Exact = 0
	CaseInsensitive                  // CaseInsensitive = 1
	Anywhere                         // Anywhere = 2
	Start                            // Start = 3
	End                              // End = 4
)

func Page[T any, S ~[]T](q *psql.ViewQuery[T, S], pg *Pagination) *psql.ViewQuery[T, S] {
	q.Apply(sm.Limit(pg.Limit), sm.Offset(pg.Offset))
	return q
}

func Order[T any, S ~[]T](q *psql.ViewQuery[T, S], s *Sort) *psql.ViewQuery[T, S] {
	if s.Direction == "asc" {
		q.Apply(sm.OrderBy(s.Column).Asc())
	}
	if s.Direction == "desc" {
		q.Apply(sm.OrderBy(s.Column).Desc())
	}
	return q
}

// ApplyFilter applies multiple filter conditions to a query.
// If no conditions are provided, it does not apply any filters.
func ApplyFilter[T any, S ~[]T](q *psql.ViewQuery[T, S], f *Filter) *psql.ViewQuery[T, S] {
	// Check if there are any conditions to apply
	if len(f.Conditions) == 0 {
		// No filter conditions, return the query as is
		return q
	}

	// Apply each filter condition
	for _, condition := range f.Conditions {
		switch condition.Mode {
		case Exact:
			q.Apply(sm.Where(psql.Quote(condition.Column).EQ(psql.Arg(condition.Value))))
		case CaseInsensitive:
			q.Apply(sm.Where(psql.Quote(condition.Column).ILike(psql.Arg(condition.Value))))
		case Start:
			q.Apply(sm.Where(psql.Quote(condition.Column).ILike(psql.Arg("%" + condition.Value))))
		case Anywhere:
			q.Apply(sm.Where(psql.Quote(condition.Column).ILike(psql.Arg("%" + condition.Value + "%"))))
		case End:
			q.Apply(sm.Where(psql.Quote(condition.Column).ILike(psql.Arg(condition.Value + "%"))))
		default:
			q.Apply(sm.Where(psql.Quote(condition.Column).ILike(psql.Arg(condition.Value))))
		}
	}
	return q
}

func ApplyAll[T any, S ~[]T](q *psql.ViewQuery[T, S], params *QueryParams) *psql.ViewQuery[T, S] {
	if params == nil { // shity but w/e
		return q
	}
	q = Page(q, &params.Pagination)
	q = Order(q, &params.Sort)
	q = ApplyFilter(q, &params.Filter)
	return q
}

func (qp QueryParams) String() string {
	return fmt.Sprintf("QueryParams: [Sort: column: %s, direction: %s], [Filter: conditions: %v], [Page: offset: %d, limit: %d]",
		qp.Column, qp.Direction,
		qp.Conditions,
		qp.Offset, qp.Limit)
}

func (c FilterCondition) String() string {
	return fmt.Sprintf("FilterCondition: [Column: %s, MatchMode: %s, Value: %s", c.Column, c.Mode, c.Value)
}

func (m MatchMode) String() string {
	switch m {
	case Exact:
		return "Exact"
	case CaseInsensitive:
		return "CaseInsensitive"
	case Start:
		return "Start"
	case Anywhere:
		return "Anywhere"
	case End:
		return "End"
	default:
		return "Unknown"
	}
}
