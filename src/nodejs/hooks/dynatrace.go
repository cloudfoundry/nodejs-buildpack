package hooks

import (
	"github.com/cloudfoundry/libbuildpack"
	"github.com/Dynatrace/libbuildpack-dynatrace"
)

func init() {
	libbuildpack.AddHook(dynatrace.NewHook("nodejs", "process"))
}
