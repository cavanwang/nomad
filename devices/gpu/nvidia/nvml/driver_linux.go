package nvml

import (
	qutil "devgitlab.lianoid.com/eaas/infra/pkg/qemu/util"
	"fmt"
	//"github.com/NVIDIA/gpu-monitoring-tools/bindings/go/nvml"
)

// Initialize nvml library by locating nvml shared object file and calling ldopen
func (n *nvmlDriver) Initialize() error {
	return nil
	//return nvml.Init()
}

// Shutdown stops any further interaction with nvml
func (n *nvmlDriver) Shutdown() error {
	return nil
	//return nvml.Shutdown()
}

// SystemDriverVersion returns installed driver version
func (n *nvmlDriver) SystemDriverVersion() (string, error) {
	return "pci-mock-1.0", nil
	//return nvml.GetDriverVersion()
}

// DeviceCount reports number of available GPU devices
func (n *nvmlDriver) DeviceCount() (uint, error) {
	pcis, err := qutil.LspciOnHost()
	if err != nil {
		return 0, err
	}
	ret := filterNvidiaGpuInfo(pcis)
	return uint(len(ret)), nil
	//return nvml.GetDeviceCount()
}

func filterNvidiaGpuInfo(pcis map[string]*qutil.PciInfo) []*qutil.PciInfo {
	tmp := []*qutil.PciInfo{}
	for _, pci := range pcis {
		if pci.Vendor == 0x10de && pci.Device == 0x102d {
			t := *pci
			tmp = append(tmp, &t)
		}
	}

	swaped := false
	for i := len(tmp) - 1; i >= 1; i-- {
		for j := 0; j <= i-1; j++ {
			if tmp[j].Slot > tmp[j+1].Slot {
				t := tmp[j]
				tmp[j] = tmp[j+1]
				tmp[j+1] = t
				swaped = true
			}
		}
		if !swaped {
			break
		}
	}
	return tmp
}

// DeviceInfoByIndex returns DeviceInfo for index GPU in system device list
func (n *nvmlDriver) DeviceInfoByIndex(index uint) (*DeviceInfo, error) {
	/*device, err := nvml.NewDevice(index)
	if err != nil {
		return nil, err
	}
	deviceMode, err := device.GetDeviceMode()
	if err != nil {
		return nil, err
	} */
	pcis, err := qutil.LspciOnHost()
	if err != nil {
		return nil, err
	}
	sortedPcis := filterNvidiaGpuInfo(pcis)
	if int(index) > len(sortedPcis)-1 {
		return nil, fmt.Errorf("index '%d' out of nvidia device range", index)
	}
	//	"UUID":"GPU-915f1d6a-81e4-81df-4d1b-20ef8bfe3503","PCIBusID":"00000000:06:00.0","DisplayState":"Enabled","PersistenceMode":"Enabled","Name":"Tesla K80","MemoryMiB":11441,"PowerW":149,"BAR1MiB":16384,"PCIBandwidthMBPerS":15760,"CoresClockMHz":875,"MemoryClockMHz":2505}
	memMb := uint64(11441)
	powerW := uint(149)
	bar1Mb := uint64(16384)
	pciBandwith := uint(15760)
	coreClock := uint(875)
	memClock := uint(2505)
	return &DeviceInfo{
		UUID:               fmt.Sprintf("%d", index),
		Name:               &sortedPcis[index].DeviceDesc,
		MemoryMiB:          &memMb,
		PowerW:             &powerW,
		BAR1MiB:            &bar1Mb,
		PCIBandwidthMBPerS: &pciBandwith,
		PCIBusID:           sortedPcis[index].Slot,
		CoresClockMHz:      &coreClock,
		MemoryClockMHz:     &memClock,
		DisplayState:       "Enabled",
		PersistenceMode:    "Enabled",
	}, nil

	/*return &DeviceInfo{
		UUID:               device.UUID,
		Name:               device.Model,
		MemoryMiB:          device.Memory,
		PowerW:             device.Power,
		BAR1MiB:            device.PCI.BAR1,
		PCIBandwidthMBPerS: device.PCI.Bandwidth,
		PCIBusID:           device.PCI.BusID,
		CoresClockMHz:      device.Clocks.Cores,
		MemoryClockMHz:     device.Clocks.Memory,
		DisplayState:       deviceMode.DisplayInfo.Mode.String(),
		PersistenceMode:    deviceMode.Persistence.String(),
	}, nil
	*/
}

// DeviceInfoByIndex returns DeviceInfo and DeviceStatus for index GPU in system device list
func (n *nvmlDriver) DeviceInfoAndStatusByIndex(index uint) (*DeviceInfo, *DeviceStatus, error) {
	/*device, err := nvml.NewDevice(index)
	if err != nil {
		return nil, nil, err
	}
	status, err := device.Status()
	if err != nil {
		return nil, nil, err
	} */

	ret1, err := n.DeviceInfoByIndex(index)
	if err != nil {
		return nil, nil, err
	}

	// {"PowerUsageW":30,"TemperatureC":42,"GPUUtilization":0,"MemoryUtilization":0,"EncoderUtilization":0,"DecoderUtilization":0,"BAR1UsedMiB":2,"UsedMemoryMiB":0,"ECCErrorsL1Cache":0,"ECCErrorsL2Cache":0,"ECCErrorsDevice":0}
	tmpC := uint(42)
	gpuUtil := uint(0)
	memUtil := uint(0)
	encoderUtil := uint(0)
	decoderUtil := uint(0)

	powerUsage := uint(30)
	bar1UsedMb := uint64(2)
	zero1 := uint64(0)
	zero2 := uint64(0)
	zero3 := uint64(0)
	zero4 := uint64(0)

	ret2 := &DeviceStatus{
		TemperatureC:       &tmpC,
		GPUUtilization:     &gpuUtil,
		MemoryUtilization:  &memUtil,
		EncoderUtilization: &encoderUtil,
		DecoderUtilization: &decoderUtil,

		UsedMemoryMiB:    &zero1,
		ECCErrorsL1Cache: &zero2,
		ECCErrorsL2Cache: &zero3,
		ECCErrorsDevice:  &zero4,
		PowerUsageW:      &powerUsage,
		BAR1UsedMiB:      &bar1UsedMb,
	}
	return ret1, ret2, nil
}
