package vm

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/bytearena/bytearena/common/utils"
	"github.com/rkt/rkt/networking/tuntap"
	// "github.com/vishvananda/netlink"
)

type VMConfig struct {
	Name          string
	ImageLocation string
	MegMemory     int
	CPUAmount     int
	CPUCoreAmount int
}

type VM struct {
	Config    VMConfig
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	stderr    io.ReadCloser
	process   *os.Process
	tapIfName string
	brIfName  string
}

func NewVM(config VMConfig) *VM {
	return &VM{
		Config:   config,
		brIfName: "docker0",
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
	err := tuntap.RemovePersistentIface(vm.tapIfName, tuntap.Tap)

	if err != nil {
		return err
	}

	vm.Log(fmt.Sprintf("Teardown network %s<->%s", vm.brIfName, vm.tapIfName))

	return nil
}

func (vm *VM) SetupNetwork() (string, error) {
	tapIfce, tapErr := createTapInterface(vm.Config.Name)
	utils.Check(tapErr, "Could not create tap interface")

	tapLinkErr := createTapLink(tapIfce)
	utils.Check(tapLinkErr, "Could not create tap link")

	go listenTap(tapIfce)

	vm.Log(fmt.Sprintf("Setup network %s<->%s", vm.brIfName, tapIfce.Name()))

	return tapIfce.Name(), nil
}

func (vm *VM) Start() error {
	kvmbin, err := exec.LookPath("kvm")

	if err != nil {
		return errors.New("Error: kvm not found in $PATH")
	}

	tapName, netErr := vm.SetupNetwork()
	utils.Check(netErr, "Could not setup VM network")

	cmd := exec.Command(
		kvmbin,
		"-name", vm.Config.Name,
		"-m", strconv.Itoa(vm.Config.MegMemory)+"M",
		"-snapshot",
		"-smp", strconv.Itoa(vm.Config.CPUAmount)+",cores="+strconv.Itoa(vm.Config.CPUCoreAmount),
		"-nographic",
		"-no-fd-bootchk",
		"-net", "nic,model=virtio",
		"-net", "user",
		"-net", "tap,name=net0,ifname="+tapName+",script=no",
		"-drive", "file="+vm.Config.ImageLocation+",if=virtio,cache=none,format=raw,index=1",
	)

	cmd.Env = nil

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
	config := VMConfig{
		Name:          "testvm",
		MegMemory:     1024,
		CPUAmount:     1,
		CPUCoreAmount: 1,
		ImageLocation: "/home/sven/go/src/github.com/bytearena/linuxkit/linuxkit.raw",
	}

	vm := NewVM(config)

	startErr := vm.Start()
	utils.Check(startErr, "Could not start VM")

	<-time.After(20 * time.Second)
	vm.SendStdin("ifconfig")
	vm.SendStdin("route -n")
	vm.SendStdin("ping 8.8.8.8 -W 3 -w 3")

	<-time.After(30 * time.Second)

	if haltErr := vm.SendHalt(); haltErr != nil {
		vm.Log(haltErr.Error())

		vm.KillProcess()
	}
}
