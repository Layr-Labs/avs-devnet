package kurtosis

import (
	"github.com/kurtosis-tech/kurtosis/api/golang/engine/lib/kurtosis_context"
)

func InitKurtosisContext() (*kurtosis_context.KurtosisContext, error) {
	return kurtosis_context.NewKurtosisContextFromLocalEngine()
}
