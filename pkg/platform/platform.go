package platform

import (
	"fmt"

	"github.com/samber/lo"
)

type PlatformErrors struct {
	Errors []PlatformError `json:"errors"`
}

func (e PlatformErrors) String() string {
	return lo.Reduce(
		e.Errors,
		func(agg string, err PlatformError, _ int) string {
			if agg == "" {
				return err.Message
			}

			return fmt.Sprintf("%s %s.", agg, err.Message)
		},
		"",
	)
}

type PlatformError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
