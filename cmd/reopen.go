package cmd

import (
	"github.com/svandragt/park/internal/park"
)

func RunReopen(store *park.Store, args []string) error {
	return setStatus(store, args, "active")
}
