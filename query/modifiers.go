package query

import (
	"context"

	"github.com/tschuyebuhl/aids/userctx"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

func UserIDModifier(ctx context.Context) bob.Mod[*dialect.SelectQuery] {
	return sm.Where(psql.Quote("user_id").EQ(psql.Arg(userctx.MustUserID(ctx))))
}

// NOTE: this is essentially unsafe.
func HabitCodeModifier(ctx context.Context) bob.Mod[*dialect.SelectQuery] {
	return sm.Where(psql.Quote("habit_code").EQ(psql.Arg(ctx.Value("habit_code"))))
}
