package data

import (
	"fmt"
	"math/rand/v2"
	"time"
)

func Code(name string) string {
	//nolint:gosec,G404
	return fmt.Sprintf("%d-%s-%d", (rand.IntN(99999)+10000)%100000, Slugify(name), time.Now().Year())
}
