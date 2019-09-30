package vfio

import (
	"context"
	"fmt"
	"time"

	//"github.com/hashicorp/nomad/devices/gpu/nvidia/nvml"
	"github.com/hashicorp/nomad/helper"
	"github.com/hashicorp/nomad/plugins/device"
	"github.com/hashicorp/nomad/plugins/shared/structs"

	qutil "devgitlab.lianoid.com/eaas/infra/pkg/qemu/util"
	"encoding/json"
)

const (
	// Attribute names and units for reporting Fingerprint output
	DriverVersionAttr = "driver_version"
)

type FingerprintDeviceData struct {
	UUID       string // SlotNum
	DeviceName *string
}

// FingerprintData represets attributes of driver/devices
type FingerprintData struct {
	Devices       []*FingerprintDeviceData
	DriverVersion string
}

// fingerprint is the long running goroutine that detects hardware
func (d *VfioGpuDevice) fingerprint(ctx context.Context, devices chan<- *device.FingerprintResponse) {
	defer close(devices)

	if d.initErr != nil {
		//if d.initErr.Error() != nvml.UnavailableLib.Error() {
		d.logger.Error("exiting fingerprinting due to problems with init loading", "error", d.initErr)
		devices <- device.NewFingerprintError(d.initErr)
		//	}

		// Just close the channel to let server know that there are no working
		// Nvidia GPU units
		return
	}

	// Create a timer that will fire immediately for the first detection
	ticker := time.NewTimer(0)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ticker.Reset(d.fingerprintPeriod)
		}
		d.writeFingerprintToChannel(devices)
	}
}

// writeFingerprintToChannel makes nvml call and writes response to channel
func (d *VfioGpuDevice) writeFingerprintToChannel(devices chan<- *device.FingerprintResponse) {
	fingerprintData, err := getFingerprintData()
	if err != nil {
		d.logger.Error("failed to get fingerprint vfio-pci gpu devices", "error", err)
		devices <- device.NewFingerprintError(err)
		return
	}

	by, _ := json.Marshal(fingerprintData)
	fmt.Printf("got fingerprint data is --------------------------------------%s\n", string(by))

	// check if any device health was updated or any device was added to host
	if !d.fingerprintChanged(fingerprintData.Devices) {
		fmt.Println("fingerprint data not changed --------------------------------------")
		return
	}

	commonAttributes := map[string]*structs.Attribute{
		DriverVersionAttr: {
			String: helper.StringToPtr(fingerprintData.DriverVersion),
		},
	}

	// Group all FingerprintDevices by DeviceName attribute
	deviceListByDeviceName := make(map[string][]*FingerprintDeviceData)
	for _, device := range fingerprintData.Devices {
		deviceName := device.DeviceName
		if deviceName == nil {
			// If it was not able to detect device name.
			// we placed them to single group with 'notAvailable' name
			notAvailableCopy := notAvailable
			deviceName = &notAvailableCopy
		}

		deviceListByDeviceName[*deviceName] = append(deviceListByDeviceName[*deviceName], device)
	}

	// Build Fingerprint response with computed groups and send it over the channel
	deviceGroups := make([]*device.DeviceGroup, 0, len(deviceListByDeviceName))
	for groupName, devices := range deviceListByDeviceName {
		deviceGroups = append(deviceGroups, deviceGroupFromFingerprintData(groupName, devices, commonAttributes))
	}
	by, _ = json.Marshal(deviceGroups)
	fmt.Printf("response fingerprint is-----------------------------------------%s\n", string(by))
	devices <- device.NewFingerprint(deviceGroups...)
}

// fingerprintChanged checks if there are any previously unseen nvidia devices located
// or any of fingerprinted nvidia devices disappeared since the last fingerprint run.
// Also, this func updates device map on VfioGpuDevice with the latest data
func (d *VfioGpuDevice) fingerprintChanged(allDevices []*FingerprintDeviceData) bool {
	d.deviceLock.Lock()
	defer d.deviceLock.Unlock()

	fingerprintDeviceMap := make(map[string]struct{})
	for _, device := range allDevices {
		fingerprintDeviceMap[device.UUID] = struct{}{}
	}

	if len(fingerprintDeviceMap) != len(d.devices) {
		d.devices = fingerprintDeviceMap
		return true
	}

	changeDetected := false
	// check if every device in d.devices is in allDevices
	for id := range d.devices {
		if _, ok := fingerprintDeviceMap[id]; !ok {
			changeDetected = true
			break
		}
	}

	d.devices = fingerprintDeviceMap
	return changeDetected
}

// deviceGroupFromFingerprintData composes deviceGroup from FingerprintDeviceData slice
func deviceGroupFromFingerprintData(groupName string, deviceList []*FingerprintDeviceData, commonAttributes map[string]*structs.Attribute) *device.DeviceGroup {
	// deviceGroup without devices makes no sense -> return nil when no devices are provided
	if len(deviceList) == 0 {
		return nil
	}

	devices := make([]*device.Device, len(deviceList))
	for index, dev := range deviceList {
		devices[index] = &device.Device{
			ID: dev.UUID,
			// all fingerprinted devices are "healthy" for now
			// to get real health data -> dcgm bindings should be used
			Healthy: true,
			HwLocality: &device.DeviceLocality{
				PciBusID: dev.UUID,
			},
		}
	}

	deviceGroup := &device.DeviceGroup{
		Vendor:  vendor,
		Type:    deviceType,
		Name:    groupName,
		Devices: devices,
		// Assumption made that devices with the same DeviceName have the same
		// attributes like amount of memory, power, bar1memory etc
		Attributes: attributesFromFingerprintDeviceData(deviceList[0]),
	}

	// Extend attribute map with common attributes
	for attributeKey, attributeValue := range commonAttributes {
		deviceGroup.Attributes[attributeKey] = attributeValue
	}

	return deviceGroup
}

// attributesFromFingerprintDeviceData converts nvml.FingerprintDeviceData
// struct to device.DeviceGroup.Attributes format (map[string]string)
// this function performs all nil checks for FingerprintDeviceData pointers
func attributesFromFingerprintDeviceData(d *FingerprintDeviceData) map[string]*structs.Attribute {
	attrs := map[string]*structs.Attribute{}
	return attrs
}

func getFingerprintData() (*FingerprintData, error) {
	pcis, err := qutil.LspciOnHost()
	if err != nil {
		return nil, err
	}

	// vfio-pci managed GPUs
	pci_list := filterGpuManagedByVfio(pcis)
	version, err := getDriverVersion(vfioDriver)
	if err != nil {
		return nil, err
	}

	ret := &FingerprintData{
		DriverVersion: version,
	}

	ret.Devices = []*FingerprintDeviceData{}
	for _, pci := range pci_list {
		ret.Devices = append(ret.Devices, &FingerprintDeviceData{
			DeviceName: &pci.DeviceDesc,
			UUID:       pci.Slot,
		})
	}
	return ret, nil
}

func filterGpuManagedByVfio(pcis map[string]*qutil.PciInfo) []*qutil.PciInfo {
	tmp := []*qutil.PciInfo{}
	for _, pci := range pcis {
		if pci.Vendor == 0x10de && pci.Device == 0x102d && pci.Driver == vfioDriver {
			t := *pci
			tmp = append(tmp, &t)
		}
	}
	return tmp
}
