package resources

import (
	"github.com/intel/sriov-network-device-plugin/pkg/types"
	"github.com/intel/sriov-network-device-plugin/pkg/types/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DeviceSelectors", func() {
	Describe("vendor selector", func() {
		Context("initializing", func() {
			It("should populate vendors array", func() {
				vendors := []string{"8086", "15b3"}
				sel := newVendorSelector(vendors).(*vendorSelector)
				Expect(sel.vendors).To(ConsistOf(vendors))
			})
		})
		Context("filtering", func() {
			It("should return devices matching vendor ID", func() {
				vendors := []string{"8086"}
				sel := newVendorSelector(vendors).(*vendorSelector)

				dev0 := mocks.PciNetDevice{}
				dev0.On("GetVendor").Return("8086")
				dev1 := mocks.PciNetDevice{}
				dev1.On("GetVendor").Return("15b3")

				in := []types.PciNetDevice{&dev0, &dev1}
				filtered := sel.Filter(in)

				Expect(filtered).To(ContainElement(&dev0))
				Expect(filtered).NotTo(ContainElement(&dev1))
			})
		})
	})
	Describe("device selector", func() {
		Context("initializing", func() {
			It("should populate devices array", func() {
				devices := []string{"10ed", "154c"}
				sel := newDeviceSelector(devices).(*deviceSelector)
				Expect(sel.devices).To(ConsistOf(devices))
			})
		})
		Context("filtering", func() {
			It("should return devices matching device code", func() {
				devices := []string{"10ed"}
				sel := newDeviceSelector(devices).(*deviceSelector)

				dev0 := mocks.PciNetDevice{}
				dev0.On("GetDeviceCode").Return("10ed")
				dev1 := mocks.PciNetDevice{}
				dev1.On("GetDeviceCode").Return("154c")

				in := []types.PciNetDevice{&dev0, &dev1}
				filtered := sel.Filter(in)

				Expect(filtered).To(ContainElement(&dev0))
				Expect(filtered).NotTo(ContainElement(&dev1))
			})
		})
	})
	Describe("driver selector", func() {
		Context("initializing", func() {
			It("should populate drivers array", func() {
				drivers := []string{"vfio-pci", "igb_uio"}
				sel := newDriverSelector(drivers).(*driverSelector)
				Expect(sel.drivers).To(ConsistOf(drivers))
			})
		})
		Context("filtering", func() {
			It("should return devices matching driver name", func() {
				drivers := []string{"vfio-pci"}
				sel := newDriverSelector(drivers).(*driverSelector)

				dev0 := mocks.PciNetDevice{}
				dev0.On("GetDriver").Return("vfio-pci")
				dev1 := mocks.PciNetDevice{}
				dev1.On("GetDriver").Return("i40evf")

				in := []types.PciNetDevice{&dev0, &dev1}
				filtered := sel.Filter(in)

				Expect(filtered).To(ContainElement(&dev0))
				Expect(filtered).NotTo(ContainElement(&dev1))
			})
		})
	})
	Describe("pfName selector", func() {
		Context("initializing", func() {
			It("should populate ifnames array", func() {
				pfNames := []string{"ens0", "eth0"}
				sel := newPfNameSelector(pfNames).(*pfNameSelector)
				Expect(sel.pfNames).To(ConsistOf(pfNames))
			})
		})
		Context("filtering", func() {
			It("should return devices matching interface PF name", func() {
				netDevs := []string{"ens0"}
				sel := newPfNameSelector(netDevs).(*pfNameSelector)

				dev0 := mocks.PciNetDevice{}
				dev0.On("GetPFName").Return("ens0")
				dev1 := mocks.PciNetDevice{}
				dev1.On("GetPFName").Return("eth0")

				in := []types.PciNetDevice{&dev0, &dev1}
				filtered := sel.Filter(in)

				Expect(filtered).To(ContainElement(&dev0))
				Expect(filtered).NotTo(ContainElement(&dev1))
			})
		})
	})

	Describe("linkType selector", func() {
		Context("initializing", func() {
			It("should populate linkTypes array", func() {
				linkTypes := []string{"ether"}
				sel := newLinkTypeSelector(linkTypes).(*linkTypeSelector)
				Expect(sel.linkTypes).To(ConsistOf(linkTypes))
			})
		})
		Context("filtering", func() {
			It("should return devices matching the correct link type", func() {
				linkTypes := []string{"ether"}
				sel := newLinkTypeSelector(linkTypes).(*linkTypeSelector)

				dev0 := mocks.PciNetDevice{}
				dev0.On("GetLinkType").Return("ether")
				dev1 := mocks.PciNetDevice{}
				dev1.On("GetLinkType").Return("infiniband")

				in := []types.PciNetDevice{&dev0, &dev1}
				filtered := sel.Filter(in)

				Expect(filtered).To(ContainElement(&dev0))
				Expect(filtered).NotTo(ContainElement(&dev1))
			})
		})
	})
})
