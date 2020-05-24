package rpc

import (
	"fmt"
)

// GetFullName get the full name of program includes group and name
func (x *ProcessInfo) GetFullName() string {
	if len(x.Group) > 0 {
		return fmt.Sprintf("%s:%s", x.Group, x.Name)
	}
	return x.Name
}
