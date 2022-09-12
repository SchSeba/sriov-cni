package utils

import (
	"fmt"
	"github.com/containernetworking/plugins/pkg/ns"
	"os"
	"path/filepath"
)

type PCIAllocation interface {
	SaveAllocatedPCI(string, string) error
	CleanAllocatedPCI(string) error
	IsAllocated(string) error
}

type PCIAllocator struct {
	dataDir string
}

func NewPCIAllocator(dataDir string) PCIAllocation {
	return &PCIAllocator{dataDir: filepath.Join(dataDir, "pci")}
}

// SaveAllocatedPCI creates a file with the pci address as a name and the network namespace as the content
// return error if the file was not created
func (p *PCIAllocator) SaveAllocatedPCI(pciAddress, ns string) error {
	if err := os.MkdirAll(p.dataDir, 0600); err != nil {
		return fmt.Errorf("failed to create the sriov data directory(%q): %v", p.dataDir, err)
	}

	path := filepath.Join(p.dataDir, pciAddress)
	err := os.WriteFile(path, []byte(ns), 0600)
	if err != nil {
		return fmt.Errorf("failed to write used PCI address lock file in the path(%q): %v", path, err)
	}

	return err
}

// CleanAllocatedPCI Remove the allocated PCI file
// return error if the file doesn't exist
func (p *PCIAllocator) CleanAllocatedPCI(pciAddress string) error {
	path := filepath.Join(p.dataDir, pciAddress)
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("error removing PCI address lock file %s: %v", path, err)
	}
	return nil
}

// IsAllocated checks if the PCI address file exist
// if it exists we also check the network namespace still exist if not we clean the allocation
// The function will return an error if the pci is still allocated to a running pod
func (p *PCIAllocator) IsAllocated(pciAddress string) error {
	path := filepath.Join(p.dataDir, pciAddress)
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("failed to check for pci address file for %s: %v", path, err)
	}

	dat, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read for pci address file for %s: %v", path, err)
	}

	// To prevent a locking of a PCI address for every pciAddress file we also add the netns path where it's been used
	// This way if for some reason the cmdDel command was not called but the pod namespace doesn't exist anymore
	// we release the PCI address
	_, err = ns.GetNS(string(dat))
	if err != nil {
		err = p.CleanAllocatedPCI(pciAddress)
		if err != nil {
			return fmt.Errorf("error cleaning the pci allocation for vf pci address %s: %v", pciAddress, err)
		}
	} else {
		return fmt.Errorf("error the deivce is already allocated for pci address %s", pciAddress)
	}

	return nil
}
