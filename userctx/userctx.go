package userctx

import "context"

type userIDKey struct{}

var UserIDKey = userIDKey{}

func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, UserIDKey, id)
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(UserIDKey)
	s, ok := v.(string)
	return s, ok
}

func MustUserID(ctx context.Context) string {
	id, ok := UserIDFromContext(ctx)
	if !ok {
		panic("user id missing from context")
	}
	return id
}
