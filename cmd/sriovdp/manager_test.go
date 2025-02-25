package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/intel/sriov-network-device-plugin/pkg/resources"
	"github.com/intel/sriov-network-device-plugin/pkg/types"
	fake "github.com/intel/sriov-network-device-plugin/pkg/types/mocks"
	"github.com/intel/sriov-network-device-plugin/pkg/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

func assertShouldFail(err error, shouldFail bool) {
	if shouldFail {
		Expect(err).To(HaveOccurred())
	} else {
		Expect(err).NotTo(HaveOccurred())
	}
}

var _ = Describe("Resource manager", func() {
	var (
		cp *cliParams
		rm *resourceManager
	)
	Describe("reading config", func() {
		BeforeEach(func() {
			cp = &cliParams{
				configFile:     "/tmp/sriovdp/test_config",
				resourcePrefix: "test_",
			}
			rm = newResourceManager(cp)
		})
		Context("when there's an error reading file", func() {
			BeforeEach(func() {
				os.RemoveAll("/tmp/sriovdp")
			})
			It("should fail", func() {
				err := rm.readConfig()
				Expect(err).To(HaveOccurred())
			})
		})
		Context("when there's an error unmarshalling config", func() {
			BeforeEach(func() {
				err := os.MkdirAll("/tmp/sriovdp", 0755)
				if err != nil {
					panic(err)
				}
				ioutil.WriteFile("/tmp/sriovdp/test_config", []byte("junk"), 0644)
			})
			AfterEach(func() {
				err := os.RemoveAll("/tmp/sriovdp")
				if err != nil {
					panic(err)
				}
				rm = nil
				cp = nil
			})
			It("should fail", func() {
				err := rm.readConfig()
				Expect(err).To(HaveOccurred())
			})
		})
		Context("when config reading is successful", func() {
			var err error
			BeforeEach(func() {
				// add err handling
				testErr := os.MkdirAll("/tmp/sriovdp", 0755)
				if testErr != nil {
					panic(testErr)
				}
				testErr = ioutil.WriteFile("/tmp/sriovdp/test_config", []byte(`{
						"resourceList": [{
								"resourceName": "intel_sriov_netdevice",
								"isRdma": false,
								"selectors": {
									"vendors": ["8086"],
									"devices": ["154c", "10ed"],
									"drivers": ["i40evf", "ixgbevf"]
								}
							},
							{
								"resourceName": "intel_sriov_dpdk",
								"selectors": {
									"vendors": ["8086"],
									"devices": ["154c", "10ed"],
									"drivers": ["vfio-pci"]
								}
							}
						]
					}`), 0644)
				if testErr != nil {
					panic(testErr)
				}
				err = rm.readConfig()
			})
			AfterEach(func() {
				testErr := os.RemoveAll("/tmp/sriovdp")
				if testErr != nil {
					panic(testErr)
				}
				rm = nil
				cp = nil
			})
			It("shouldn't fail", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("should load resources list", func() {
				Expect(len(rm.configList)).To(Equal(2))
			})
		})
	})
	Describe("validating configuration", func() {
		var fs *utils.FakeFilesystem
		BeforeEach(func() {
			cp = &cliParams{
				configFile:     "/tmp/sriovdp/test_config",
				resourcePrefix: "test_",
			}
			rm = newResourceManager(cp)
			fs = &utils.FakeFilesystem{
				Dirs: []string{"sys/bus/pci/devices/0000:02:00.0", "sys/bus/pci/devices/0000:03:00.0"},
				Files: map[string][]byte{
					"sys/bus/pci/devices/0000:02:00.0/sriov_numvfs": []byte("32"),
					"sys/bus/pci/devices/0000:03:00.0/sriov_numvfs": []byte("0"),
				},
			}
		})
		AfterEach(func() {
			err := os.RemoveAll("/tmp/sriovdp")
			if err != nil {
				panic(err)
			}
			rm = nil
			cp = nil
		})
		Context("when resource name is invalid", func() {
			BeforeEach(func() {
				err := os.MkdirAll("/tmp/sriovdp", 0755)
				if err != nil {
					panic(err)
				}
				err = ioutil.WriteFile("/tmp/sriovdp/test_config", []byte(`{
					"resourceList":	[{
						"resourceName": "invalid-name",
						"isRdma": false,
						"selectors": {
							"vendors": ["8086"],
							"devices": ["154c", "10ed"],
							"drivers": ["i40evf", "ixgbevf"]
						}
					}]
				}`), 0644)
				if err != nil {
					panic(err)
				}
				rm.readConfig()
			})
			It("should return false", func() {
				defer fs.Use()()
				Expect(rm.validConfigs()).To(Equal(false))
			})
		})
		Context("when resource name is duplicated", func() {
			BeforeEach(func() {
				err := os.MkdirAll("/tmp/sriovdp", 0755)
				if err != nil {
					panic(err)
				}
				err = ioutil.WriteFile("/tmp/sriovdp/test_config", []byte(`{
					"resourceList":	[{
						"resourceName": "duplicate",
						"isRdma": true,
						"selectors": {
							"vendors": ["8086"],
							"devices": ["154c", "10ed"],
							"drivers": ["i40evf", "ixgbevf"]
						}
					},{
						"resourceName": "duplicate",
						"selectors": {
							"vendors": ["8086"],
							"devices": ["154c", "10ed"],
							"drivers": ["vfio-pci"]
						}
					}]
				}`), 0644)
				if err != nil {
					panic(err)
				}
				rm.readConfig()
			})
			It("should return false", func() {
				defer fs.Use()()
				Expect(rm.validConfigs()).To(Equal(false))
			})
		})
		Describe("managing resources servers", func() {
			var rm *resourceManager
			var mockedRf *fake.ResourceFactory
			BeforeEach(func() {
				mockedRf = &fake.ResourceFactory{}
				rm = &resourceManager{
					rFactory: mockedRf,
					configList: []*types.ResourceConfig{
						&types.ResourceConfig{ResourceName: "fake"},
					},
					resourceServers: []types.ResourceServer{},
					netDeviceList:   []types.PciNetDevice{},
				}
			})
			Describe("initializing servers", func() {
				Context("when getting resource server fails", func() {
					It("should return an error", func() {
						mockedRf.
							On("GetResourcePool", rm.configList[0], rm.netDeviceList).
							Return(&fake.ResourcePool{}, nil).
							On("GetResourceServer", &fake.ResourcePool{}).
							Return(&fake.ResourceServer{}, fmt.Errorf("fake error"))

						Expect(rm.initServers()).To(HaveOccurred())
					})
				})
				Context("when initializing server fails", func() {
					var (
						mockedServer *fake.ResourceServer
					)
					BeforeEach(func() {
						mockedServer = &fake.ResourceServer{}
						mockedServer.On("Init").Return(fmt.Errorf("fake error"))
						mockedRf.
							On("GetResourcePool", rm.configList[0], rm.netDeviceList).
							Return(&fake.ResourcePool{}, nil).
							On("GetResourceServer", &fake.ResourcePool{}).
							Return(mockedServer, nil)
					})
					It("should not return an error", func() {
						Expect(rm.initServers()).NotTo(HaveOccurred())
					})
					It("should finish with empty list of servers", func() {
						Expect(len(rm.resourceServers)).To(Equal(0))
					})
				})
				Context("when server is properly initialized", func() {
					var (
						mockedServer *fake.ResourceServer
					)
					BeforeEach(func() {
						mockedServer = &fake.ResourceServer{}
						mockedRf.
							On("GetResourcePool", rm.configList[0], rm.netDeviceList).
							Return(&fake.ResourcePool{}, nil).
							On("GetResourceServer", &fake.ResourcePool{}).
							Return(mockedServer, nil)
						mockedServer.On("Init").Return(nil)
					})
					It("should not return an error", func() {
						Expect(rm.initServers()).NotTo(HaveOccurred())
					})
					It("should call Init() method on the server without getting errors", func() {
						Expect(mockedServer.MethodCalled("Init")).To(Equal(mock.Arguments{nil}))
					})
					PIt("should end up with one element in the list of servers", func() {
						Expect(rm.resourceServers).To(ContainElement(mockedServer))
					})
				})
			})
		})
	})
	DescribeTable("checking whether device is in use",
		func(fs *utils.FakeFilesystem, addr string, expected, shouldFail bool) {
			defer fs.Use()()

			actual, err := isInUse(addr)
			Expect(actual).To(Equal(expected))
			assertShouldFail(err, shouldFail)
		},
		Entry("device doesn't exist", &utils.FakeFilesystem{}, "0000:00:00.0", false, true),
		Entry("has interface in sys fs but netlink lib returns nil",
			&utils.FakeFilesystem{Dirs: []string{"sys/bus/pci/devices/0000:00:00.0/net/invalid0"}},
			"0000:00:00.0",
			true, false,
		),
	)
	DescribeTable("discovering devices",
		func(fs *utils.FakeFilesystem) {
			defer fs.Use()()
			os.Setenv("GHW_CHROOT", fs.RootDir)
			defer os.Unsetenv("GHW_CHROOT")

			rf := resources.NewResourceFactory("fake", "fake", true)
			rm := &resourceManager{
				rFactory: rf,
				configList: []*types.ResourceConfig{
					&types.ResourceConfig{ResourceName: "fake"},
				},
				resourceServers: []types.ResourceServer{},
				netDeviceList:   []types.PciNetDevice{},
			}

			err := rm.discoverHostDevices()
			Expect(err).NotTo(HaveOccurred())
		},
		Entry("no devices",
			&utils.FakeFilesystem{
				Dirs: []string{"sys/bus/pci/devices"},
			},
		),
		Entry("unparsable modalias",
			&utils.FakeFilesystem{
				Dirs: []string{
					"sys/bus/pci/devices/0000:00:00.0",
					"sys/bus/pci/drivers/i40e",
				},
				Files: map[string][]byte{
					"sys/bus/pci/devices/0000:00:00.0/modalias":       []byte("pci:junk"),
					"sys/bus/pci/devices/0000:00:00.0/sriov_totalvfs": []byte("0"),
				},
				Symlinks: map[string]string{
					"sys/bus/pci/devices/0000:00:00.0/driver": "../../../../bus/pci/drivers/i40e",
				},
			},
		),
		Entry("PF device with no VFs configured",
			&utils.FakeFilesystem{
				Dirs: []string{
					"sys/bus/pci/devices/0000:00:00.0",
					"sys/bus/pci/drivers/i40e",
				},
				Files: map[string][]byte{
					"sys/bus/pci/devices/0000:00:00.0/modalias": []byte(
						"pci:v00008086d00001572sv00008086sd00000004bc02sc00i00",
					),
					"sys/bus/pci/devices/0000:00:00.0/sriov_totalvfs": []byte("0"),
				},
				Symlinks: map[string]string{
					"sys/bus/pci/devices/0000:00:00.0/driver": "../../../../bus/pci/drivers/i40e",
				},
			},
		),
		Entry("PF device with no VFs configured and not bound to any driver",
			&utils.FakeFilesystem{
				Dirs: []string{
					"sys/bus/pci/devices/0000:00:00.0",
				},
				Files: map[string][]byte{
					"sys/bus/pci/devices/0000:00:00.0/modalias": []byte(
						"pci:v00008086d00001572sv00008086sd00000004bc02sc00i00",
					),
					"sys/bus/pci/devices/0000:00:00.0/sriov_totalvfs": []byte("0"),
				},
			},
		),
		Entry("PF device with VF configured",
			&utils.FakeFilesystem{
				Dirs: []string{
					"sys/bus/pci/devices/0000:00:00.0",
					"sys/bus/pci/devices/0000:00:00.1",
					"sys/bus/pci/drivers/i40e",
					"sys/bus/pci/drivers/i40evf",
				},
				Files: map[string][]byte{
					"sys/bus/pci/devices/0000:00:00.0/modalias": []byte(
						"pci:v00008086d00001572sv00008086sd00000004bc02sc00i00",
					),
					"sys/bus/pci/devices/0000:00:00.1/modalias": []byte(
						"pci:v00008086d0000154Csv00008086sd00000000bc02sc00i00",
					),
					"sys/bus/pci/devices/0000:00:00.0/sriov_totalvfs": []byte("1"),
					"sys/bus/pci/devices/0000:00:00.0/sriov_numvfs":   []byte("1"),
				},
				Symlinks: map[string]string{
					"sys/bus/pci/devices/0000:00:00.0/driver":  "../../../../bus/pci/drivers/i40e",
					"sys/bus/pci/devices/0000:00:00.1/driver":  "../../../../bus/pci/drivers/i40evf",
					"sys/bus/pci/devices/0000:00:00.0/virtfn0": "../0000:00:00.1",
				},
			},
		),
	)
	DescribeTable("adding to link watch list",
		func(fs *utils.FakeFilesystem, addr string, expected []types.LinkWatcher) {
			defer fs.Use()()

			rf := resources.NewResourceFactory("fake", "fake", true)
			rm := &resourceManager{
				rFactory: rf,
				configList: []*types.ResourceConfig{
					&types.ResourceConfig{ResourceName: "fake"},
				},
				resourceServers: []types.ResourceServer{},
				netDeviceList:   []types.PciNetDevice{},
				linkWatchList:   make(map[string]types.LinkWatcher, 0),
			}

			rm.addToLinkWatchList(addr)
			Expect(rm.linkWatchList).To(ConsistOf(expected))
		},
		Entry("no network interfaces on the device",
			&utils.FakeFilesystem{Dirs: []string{"sys/bus/pci/devices/0000:00:00.0"}},
			"0000:00:00.0",
			[]types.LinkWatcher{},
		),
		Entry("single network interface",
			&utils.FakeFilesystem{Dirs: []string{"sys/bus/pci/devices/0000:00:00.0/net/fakenet0"}},
			"0000:00:00.0",
			[]types.LinkWatcher{&linkWatcher{ifName: "fakenet0"}},
		),
	)
})
