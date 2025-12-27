package query

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/tschuyebuhl/httpkit/userctx"
)

func TestUserIDModifier(t *testing.T) {
	ctx := userctx.WithUserID(context.Background(), "user-1")

	q := psql.Select(
		sm.Columns("*"),
		sm.From("habits"),
	)
	q.Apply(UserIDModifier(ctx))

	var buf bytes.Buffer
	args, err := q.WriteQuery(context.Background(), &buf, 1)
	if err != nil {
		t.Fatalf("write query: %v", err)
	}

	sql := buf.String()
	if !strings.Contains(sql, `WHERE ("user_id" = $1)`) {
		t.Fatalf("unexpected sql: %s", sql)
	}
	fmt.Println(sql)
	if len(args) != 1 || args[0] != "user-1" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestHabitCodeModifier(t *testing.T) {
	ctx := context.WithValue(context.Background(), "habit_code", "hc-1")

	q := psql.Select(
		sm.Columns("*"),
		sm.From("habits"),
	)
	q.Apply(HabitCodeModifier(ctx))

	var buf bytes.Buffer
	args, err := q.WriteQuery(context.Background(), &buf, 1)
	if err != nil {
		t.Fatalf("write query: %v", err)
	}

	sql := buf.String()
	if !strings.Contains(sql, `WHERE ("habit_code" = $1)`) {
		t.Fatalf("unexpected sql: %s", sql)
	}
	fmt.Println(sql)
	if len(args) != 1 || args[0] != "hc-1" {
		t.Fatalf("unexpected args: %#v", args)
	}
}
