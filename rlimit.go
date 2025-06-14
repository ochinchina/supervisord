//go:build !windows && !freebsd
// +build !windows,!freebsd

package main

import (
	"fmt"
	"syscall"
)

func (s *Supervisor) checkRequiredResources() error {
	if minfds, vErr := s.getMinRequiredRes("minfds"); vErr == nil {
		return s.checkMinLimit(syscall.RLIMIT_NOFILE, "NOFILE", minfds)
	}
	if minprocs, vErr := s.getMinRequiredRes("minprocs"); vErr == nil {
		// RPROC = 6
		return s.checkMinLimit(6, "NPROC", minprocs)
	}
	return nil

}

func (s *Supervisor) getMinRequiredRes(resourceName string) (uint64, error) {
	if entry, ok := s.config.GetSupervisord(); ok {
		value := uint64(entry.GetInt(resourceName, 0))
		if value > 0 {
			return value, nil
		} else {
			return 0, fmt.Errorf("No such key %s", resourceName)
		}
	} else {
		return 0, fmt.Errorf("No supervisord section")
	}

}

func (s *Supervisor) checkMinLimit(resource int, resourceName string, minRequiredSource uint64) error {
	var limit syscall.Rlimit

	if syscall.Getrlimit(resource, &limit) != nil {
		return fmt.Errorf("fail to get the %s limit", resourceName)
	}

	if minRequiredSource > limit.Max {
		return fmt.Errorf("%s %d is greater than Hard limit %d", resourceName, minRequiredSource, limit.Max)
	}

	if limit.Cur >= minRequiredSource {
		return nil
	}

	limit.Cur = limit.Max
	if syscall.Setrlimit(syscall.RLIMIT_NOFILE, &limit) != nil {
		return fmt.Errorf("fail to set the %s to %d", resourceName, limit.Cur)
	}
	return nil
}
