package vfio

import (
	"fmt"
	"io/ioutil"
)

// /sys/bus/pci/drivers/vfio-pci/module/version
func getDriverVersion(name string) (string, error) {
	sysPath := fmt.Sprintf("/sys/bus/pci/drivers/%s/module/version", name)
	by, err := ioutil.ReadFile(sysPath)
	if err != nil {
		return "", fmt.Errorf("read file '%s' error: %v", err)
	}
	return string(by), nil
}
