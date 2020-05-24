package rpc

import (
	"os"

	"github.com/stuartcarnie/gopm/signals"
)

func (x ProcessSignal) ToSignal() (os.Signal, error) {
	return signals.ToSignal(x.String())
}
