package vm

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/bytearena/bytearena/arenamaster/vm/types"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/rkt/rkt/networking/tuntap"
	// "github.com/vishvananda/netlink"
)

type NIC struct {
	Type    string
	Connect string
	Model   string
	Name    string
	Ifname  string
	Script  string
}

type VMConfig struct {
	NICs          []interface{}
	Name          string
	ImageLocation string
	MegMemory     int
	CPUAmount     int
	CPUCoreAmount int
}

type VM struct {
	Config  VMConfig
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	stderr  io.ReadCloser
	process *os.Process
}

func NewVM(config VMConfig) *VM {
	return &VM{
		Config: config,
	}
}

func (vm *VM) readStdout(reader io.Reader) {
	buffReader := bufio.NewReader(reader)

	for {
		line, _, readErr := buffReader.ReadLine()

		if readErr == io.EOF {
			break
		}

		if len(line) == 0 {
			continue
		}

		vm.Log(string(line))
	}
}

func (vm *VM) Log(msg string) {
	fmt.Printf("[%s] %s\n", vm.Config.Name, msg)
}

func (vm *VM) SendStdin(command string) error {
	if vm.stdin == nil {
		return errors.New("Could not send halt: stdin not available")
	}

	_, err := vm.stdin.Write([]byte(command + "\n"))

	if err != nil {
		return err
	}

	return nil
}

func (vm *VM) SendHalt() error {
	vm.Log("Halting...")

	return vm.SendStdin("halt")
}

func (vm *VM) KillProcess() error {
	vm.Log("Killing process...")

	if vm.process == nil {
		return errors.New("Could not kill process: process not available")
	}

	vm.process.Kill()

	return nil
}

func (vm *VM) Close() {
	var closeErr error

	closeErr = vm.stdout.Close()
	utils.Check(closeErr, "Could not close stdout")

	closeErr = vm.stderr.Close()
	utils.Check(closeErr, "Could not close stderr")

	closeErr = vm.stdin.Close()
	utils.Check(closeErr, "Could not close stdin")
}

func (vm *VM) TearNetwork() error {

	return nil
}

func (vm *VM) Start() error {
	kvmbin, err := exec.LookPath("kvm")

	if err != nil {
		return errors.New("Error: kvm not found in $PATH")
	}

	cmd := CreateKVMCommand(kvmbin, vm.Config)

	stdin, stdinErr := cmd.StdinPipe()
	utils.Check(stdinErr, "Could not get stdin")

	stdout, stdoutErr := cmd.StdoutPipe()
	utils.Check(stdoutErr, "Could not get stdout")

	stderr, stderrErr := cmd.StderrPipe()
	utils.Check(stderrErr, "Could not get stderr")

	vm.stdout = stdout
	vm.stderr = stderr
	vm.stdin = stdin

	vm.Log("Starting...")

	err = cmd.Start()

	if err != nil {
		return errors.New("Error: VM could not be Started: " + err.Error())
	}

	vm.process = cmd.Process

	go vm.readStdout(stdout)
	go vm.readStdout(stderr)

	go func() {
		waitErr := cmd.Wait()
		utils.Check(waitErr, "Could not wait VM process")

		vm.Log("Stopped")
		vm.Close()
	}()

	return nil
}

func Test() {
	// brIfName := "docker0"
	vmName := "testvm"

	runIP("link", "add", "br0", "type", "bridge")
	runIP("link", "set", "tap0", "master", "br0")
	runIP("link", "set", "br0", "up")
	runIP("link", "set", "tap0", "up")

	// tapIfce, tapErr := createTapInterface()
	// utils.Check(tapErr, "Could not create tap interface")

	// tapLinkErr := createTapLink(tapIfce)
	// utils.Check(tapLinkErr, "Could not create tap link")

	// // socketAddr := "127.0.0.1:1234"

	// go listenTap(tapIfce)

	// fmt.Printf("Setup network %s<->%s", brIfName, tapIfce.Name())

	config := VMConfig{
		NICs: []interface{}{
			types.NICUser{},
			types.NICIface{
				Model: "virtio",
			},
			types.NICTap{
				Name:   "net0",
				Ifname: "tap0",
				Script: "no",
			},
		},
		Name:          vmName,
		MegMemory:     1024,
		CPUAmount:     1,
		CPUCoreAmount: 1,
		ImageLocation: "/home/sven/go/src/github.com/bytearena/linuxkit/linuxkit.raw",
	}

	vm := NewVM(config)

	startErr := vm.Start()
	utils.Check(startErr, "Could not start VM")

	<-time.After(20 * time.Second)

	vm.SendStdin("echo ----------------------------------------------------------------------------------------------------")

	vm.SendStdin("ifconfig")
	vm.SendStdin("route -n")
	vm.SendStdin("ping 8.8.8.8 -W 3 -w 3")
	vm.SendStdin("ping bytearena.com -W 3 -w 3")
	// vm.SendStdin("ping 192.168.1.120 -W 3 -w 3")
	// vm.SendStdin("ifconfig eth0")

	<-time.After(30 * time.Second)

	if haltErr := vm.SendHalt(); haltErr != nil {
		vm.Log(haltErr.Error())

		killErr := vm.KillProcess()
		utils.Check(killErr, "Could not kill VM process")
	}

	ifName := "tap0"
	errRemoveTap := tuntap.RemovePersistentIface(ifName, tuntap.Tap)

	if errRemoveTap != nil {
		panic(errRemoveTap)
	}

	// vm.Log(fmt.Sprintf("Teardown network %s<->%s", brIfName, tapIfce.Name()))
}
