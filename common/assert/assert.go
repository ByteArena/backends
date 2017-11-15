package assert

import (
	"github.com/bytearena/bytearena/common/utils"
	bettererrors "github.com/xtuc/better-errors"
)

func Assert(cond bool, msg string) {

	if !cond {
		berror := bettererrors.
			NewFromString("Assertion error").
			With(bettererrors.NewFromString(msg))

		utils.FailWith(berror)
	}
}
