// +build !windows

package process

func (p *Process) postStart() error {
	return nil
}
