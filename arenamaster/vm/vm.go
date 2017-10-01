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
	QMPServer     *types.QMPServer
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
	hostIp := "10.0.2.10"
	vmName := "arenaserver-1"

	config := VMConfig{
		QMPServer: &types.QMPServer{
			Addr: "tcp:localhost:4444",
		},
		NICs: []interface{}{
			types.NICUser{
				Host:     hostIp,
				Hostname: vmName,
			},
			types.NICIface{
				Model: "virtio",
			},
		},
		Name:          vmName,
		MegMemory:     2048,
		CPUAmount:     1,
		CPUCoreAmount: 1,
		ImageLocation: "/home/sven/go/src/github.com/bytearena/linuxkit/linuxkit.raw",
	}

	vm := NewVM(config)

	startErr := vm.Start()
	utils.Check(startErr, "Could not start VM")

	<-time.After(5 * time.Second)

	vm.SendStdin("tail -f /var/log/arenaserver.*")
	// vm.SendStdin("route -n")
	// vm.SendStdin("ping 8.8.8.8 -W 3 -w 3")
	// vm.SendStdin("ping " + hostIp + " -W 3 -w 3")
	// vm.SendStdin("ping bytearena.com -W 3 -w 3")

	<-time.After(3 * time.Minute)

	if haltErr := vm.SendHalt(); haltErr != nil {
		vm.Log(haltErr.Error())

		killErr := vm.KillProcess()
		utils.Check(killErr, "Could not kill VM process")
	}
}
