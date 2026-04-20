package cmd

import (
	"fmt"
	"strconv"

	"github.com/svandragt/park/internal/park"
)

func parseID(store *park.Store, s string) (int64, error) {
	if s == "-" {
		item, err := store.GetLast()
		if err != nil {
			return 0, err
		}
		return item.ID, nil
	}
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid id: %s", s)
	}
	return id, nil
}
