package arenamaster

import (
	"strconv"
	"sync"
	"time"

	"github.com/bytearena/bytearena/arenamaster/state"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/schnapps"
	vmid "github.com/bytearena/schnapps/id"
	vmscheduler "github.com/bytearena/schnapps/scheduler"
)

func (server *Server) createScheduler(listener Listener, healthchecks *ArenaHealthCheck) *vmscheduler.Pool {
	provisionVmFn := func() *vm.VM {
		inc++
		id := inc

		vm, err := server.SpawnArena(id)
		server.state.UpdateStateAddBootingVM(id, vm)

		if err != nil {
			utils.RecoverableError("vm", "Could not start ("+strconv.Itoa(id)+"): "+err.Error())
			server.state.UpdateStateVMErrored(id)

			return nil
		} else {
			err := vm.WaitUntilBooted()

			if err != nil {
				utils.RecoverableError("vm", "Could not wait until VM is booted")
				server.state.UpdateStateVMErrored(id)
			} else {
				server.state.UpdateStateVMBooted(id)
				utils.Debug("vm", "VM ("+strconv.Itoa(id)+") booted")
			}

			// Start timer between VM running and arena idle
			// If the VM has no arena running or idle, we better halt it
			go func() {
				<-time.After(TIME_BETWEEN_VM_RUNNING_AND_ARENA_IDLE)

				status := server.state.GetStatus(vm.Config.Id)
				isIdle := status&state.STATE_IDLE_ARENA == 0
				isRunning := status&state.STATE_RUNNING_ARENA == 0

				if !isIdle && !isRunning {

					haltMsg := types.NewMQMessage(
						"arena-master",
						"halt",
					).SetPayload(types.MQPayload{
						"id": strconv.Itoa(id),
					})

					listener.arenaHalt <- *haltMsg
				}
			}()

			return vm
		}
	}

	var healthcheckFnMutex sync.Mutex
	healtcheckVmFn := func(vm *vm.VM) bool {
		healthcheckFnMutex.Lock()
		defer healthcheckFnMutex.Unlock()

		cache := healthchecks.GetCache()
		mac, found := vmid.GetVMMAC(vm)

		if !found {
			utils.RecoverableError("healthcheck", "Error during healthcheck: mac not found")
			return false
		}

		// Ignore healthcheck if the VM is currenly booting
		isBooting := server.state.GetStatus(vm.Config.Id)&state.STATE_BOOTING_VM != 0

		if isBooting {
			return true
		}

		if res, hasRes := cache[mac]; hasRes {
			return res
		} else {
			return false
		}
	}

	pool, schedulerErr := vmscheduler.NewFixedVMPool(3)

	if schedulerErr != nil {
		panic(schedulerErr)
	}

	go func() {
		events := pool.Events()

		for {
			select {
			case msg := <-events:
				switch msg := msg.(type) {
				case vmscheduler.HEALTHCHECK:
					res := healtcheckVmFn(msg.VM)

					pool.Consumer() <- vmscheduler.HEALTHCHECK_RESULT{
						VM:  msg.VM,
						Res: res,
					}
				case vmscheduler.PROVISION:
					utils.Debug("master", "Provisioning new VM")
					vm := provisionVmFn()

					pool.Consumer() <- vmscheduler.PROVISION_RESULT{vm}
				case vmscheduler.VM_UNHEALTHY:
					id := msg.VM.Config.Id
					server.state.UpdateStateVMErrored(id)

					haltMsg := types.NewMQMessage(
						"arena-master",
						"halt",
					).SetPayload(types.MQPayload{
						"id": strconv.Itoa(id),
					})

					listener.arenaHalt <- *haltMsg
				}
			}
		}
	}()

	return pool
}
