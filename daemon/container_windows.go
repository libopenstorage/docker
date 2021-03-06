// +build windows

package daemon

import (
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/daemon/execdriver"
	derr "github.com/docker/docker/errors"
	"github.com/docker/docker/volume"
	"github.com/docker/libnetwork"
)

// DefaultPathEnv is deliberately empty on Windows as the default path will be set by
// the container. Docker has no context of what the default path should be.
const DefaultPathEnv = ""

// Container holds fields specific to the Windows implementation. See
// CommonContainer for standard fields common to all containers.
type Container struct {
	CommonContainer

	// Fields below here are platform specific.
}

func killProcessDirectly(container *Container) error {
	return nil
}

func (container *Container) setupLinkedContainers() ([]string, error) {
	return nil, nil
}

func (container *Container) createDaemonEnvironment(linkedEnv []string) []string {
	// On Windows, nothing to link. Just return the container environment.
	return container.Config.Env
}

func (container *Container) initializeNetworking() error {
	return nil
}

// ConnectToNetwork connects a container to the network
func (container *Container) ConnectToNetwork(idOrName string) error {
	return nil
}

// DisconnectFromNetwork disconnects a container from, the network
func (container *Container) DisconnectFromNetwork(n libnetwork.Network) error {
	return nil
}

func (container *Container) setupWorkingDirectory() error {
	return nil
}

func populateCommand(c *Container, env []string) error {
	en := &execdriver.Network{
		Interface: nil,
	}

	parts := strings.SplitN(string(c.hostConfig.NetworkMode), ":", 2)
	switch parts[0] {
	case "none":
	case "default", "": // empty string to support existing containers
		if !c.Config.NetworkDisabled {
			en.Interface = &execdriver.NetworkInterface{
				MacAddress:   c.Config.MacAddress,
				Bridge:       c.daemon.configStore.Bridge.VirtualSwitchName,
				PortBindings: c.hostConfig.PortBindings,

				// TODO Windows. Include IPAddress. There already is a
				// property IPAddress on execDrive.CommonNetworkInterface,
				// but there is no CLI option in docker to pass through
				// an IPAddress on docker run.
			}
		}
	default:
		return derr.ErrorCodeInvalidNetworkMode.WithArgs(c.hostConfig.NetworkMode)
	}

	// TODO Windows. More resource controls to be implemented later.
	resources := &execdriver.Resources{
		CommonResources: execdriver.CommonResources{
			CPUShares: c.hostConfig.CPUShares,
		},
	}

	// TODO Windows. Further refactoring required (privileged/user)
	processConfig := execdriver.ProcessConfig{
		Privileged:  c.hostConfig.Privileged,
		Entrypoint:  c.Path,
		Arguments:   c.Args,
		Tty:         c.Config.Tty,
		User:        c.Config.User,
		ConsoleSize: c.hostConfig.ConsoleSize,
	}

	processConfig.Env = env

	var layerPaths []string
	img, err := c.daemon.graph.Get(c.ImageID)
	if err != nil {
		return derr.ErrorCodeGetGraph.WithArgs(c.ImageID, err)
	}
	for i := img; i != nil && err == nil; i, err = c.daemon.graph.GetParent(i) {
		lp, err := c.daemon.driver.Get(i.ID, "")
		if err != nil {
			return derr.ErrorCodeGetLayer.WithArgs(c.daemon.driver.String(), i.ID, err)
		}
		layerPaths = append(layerPaths, lp)
		err = c.daemon.driver.Put(i.ID)
		if err != nil {
			return derr.ErrorCodePutLayer.WithArgs(c.daemon.driver.String(), i.ID, err)
		}
	}
	m, err := c.daemon.driver.GetMetadata(c.ID)
	if err != nil {
		return derr.ErrorCodeGetLayerMetadata.WithArgs(err)
	}
	layerFolder := m["dir"]

	c.command = &execdriver.Command{
		CommonCommand: execdriver.CommonCommand{
			ID:            c.ID,
			Rootfs:        c.rootfsPath(),
			InitPath:      "/.dockerinit",
			WorkingDir:    c.Config.WorkingDir,
			Network:       en,
			MountLabel:    c.getMountLabel(),
			Resources:     resources,
			ProcessConfig: processConfig,
			ProcessLabel:  c.getProcessLabel(),
		},
		FirstStart:  !c.HasBeenStartedBefore,
		LayerFolder: layerFolder,
		LayerPaths:  layerPaths,
		Hostname:    c.Config.Hostname,
		Isolated:    c.hostConfig.Isolation.IsHyperV(),
	}

	return nil
}

// GetSize returns real size & virtual size
func (container *Container) getSize() (int64, int64) {
	// TODO Windows
	return 0, 0
}

// setNetworkNamespaceKey is a no-op on Windows.
func (container *Container) setNetworkNamespaceKey(pid int) error {
	return nil
}

// allocateNetwork is a no-op on Windows.
func (container *Container) allocateNetwork() error {
	return nil
}

func (container *Container) updateNetwork() error {
	return nil
}

func (container *Container) releaseNetwork() {
}

// appendNetworkMounts appends any network mounts to the array of mount points passed in.
// Windows does not support network mounts (not to be confused with SMB network mounts), so
// this is a no-op.
func appendNetworkMounts(container *Container, volumeMounts []volume.MountPoint) ([]volume.MountPoint, error) {
	return volumeMounts, nil
}

func (container *Container) setupIpcDirs() error {
	return nil
}

func (container *Container) unmountIpcMounts(unmount func(pth string) error) {
}

func detachMounted(path string) error {
	return nil
}

func (container *Container) ipcMounts() []execdriver.Mount {
	return nil
}

func getDefaultRouteMtu() (int, error) {
	return -1, errSystemNotSupported
}

// conditionalMountOnStart is a platform specific helper function during the
// container start to call mount.
func (container *Container) conditionalMountOnStart() error {
	// We do not mount if a Hyper-V container
	if !container.hostConfig.Isolation.IsHyperV() {
		if err := container.Mount(); err != nil {
			return err
		}
	}
	return nil
}

// conditionalUnmountOnCleanup is a platform specific helper function called
// during the cleanup of a container to unmount.
func (container *Container) conditionalUnmountOnCleanup() {
	// We do not unmount if a Hyper-V container
	if !container.hostConfig.Isolation.IsHyperV() {
		if err := container.Unmount(); err != nil {
			logrus.Errorf("%v: Failed to umount filesystem: %v", container.ID, err)
		}
	}
}
