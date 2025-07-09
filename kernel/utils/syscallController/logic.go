package utils_syscallController

import (
	"fmt"
	"log"
	"log/slog"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
	estados "github.com/sisoputnfrba/tp-golang/kernel/utils/estados"
	general "github.com/sisoputnfrba/tp-golang/kernel/utils/general"
	utils_pl "github.com/sisoputnfrba/tp-golang/kernel/utils/planificadores/planifLargo"
	utils_pm "github.com/sisoputnfrba/tp-golang/kernel/utils/planificadores/planifMedio"
)

func manejarIO(syscallIO globals.SyscallIO) {
	globals.ListaIOsMutex.Lock()
	existe := general.VerificarExistenciaIO(syscallIO.NombreIO)
	nombreIO := syscallIO.NombreIO

	//log.Print("Vino una syscall IO a ManejarIO:", syscallIO)

	logSyscalls(syscallIO.PID, "IO")

	if !existe {
		general.FinalizarProceso(syscallIO.PID)
	} else {

		// Bloqueo el proceso y le actualizo el PC
		general.ActualizarPC(syscallIO.PID, syscallIO.PC)
		//log.Print("Se quiere loquear MapaProcesos en ManejarIO")
		globals.MapaProcesosMutex.Lock()
		proceso := globals.MapaProcesos[syscallIO.PID]
		globals.MapaProcesosMutex.Unlock()
		//log.Print("Se unloquea MapaProcesos en ManejarIO")

		general.LiberarCPU(syscallIO.NombreCPU)

		// Si hay instancias libres, envio solicitud, sino agrego a la cola
		io := globals.MapaIOs[nombreIO]
		instanciaIo, pos, hayLibre := general.BuscarInstanciaIOLibre(syscallIO.NombreIO)
		if hayLibre {
			log.Print("Seleccionada IO libre: ", instanciaIo)
			instanciaIo.PidProcesoActual = syscallIO.PID
			general.EnviarSolicitudIO(instanciaIo.Handshake.IP, instanciaIo.Handshake.Puerto, syscallIO.PID, syscallIO.Tiempo)
			io.Instancias[pos] = instanciaIo
		} else {
			io.ColaProcesosEsperando = append(io.ColaProcesosEsperando, syscallIO)
		}
		utils_pm.EjecutarPlanificadorMedioPlazo(proceso, "Syscall IO")
		globals.MapaIOs[nombreIO] = io

	}

	globals.ListaIOsMutex.Unlock()
	// Motivo de Bloqueo: ## (<PID>) - Bloqueado por IO: <DISPOSITIVO_IO>
	slog.Info(fmt.Sprintf("## (%d) - Bloqueado por IO: %s", syscallIO.PID, syscallIO.NombreIO))

}

func manejarInit_Proc(syscallINIT globals.SyscallInit) {
	logSyscalls(syscallINIT.Pid_proceso, "INIT_PROC")
	utils_pl.CrearProcesoNuevo(syscallINIT.Archivo, syscallINIT.Tamanio)

	// El proceso vuelve a ejecutar
	globals.ListaCPUsMutex.Lock()
	posCpu := general.BuscarCpu(syscallINIT.Nombre_CPU)
	cpu := globals.ListaCPUs[posCpu]
	globals.ListaCPUsMutex.Unlock()
	general.EnviarProcesoAEjecutar_ACPU(cpu.Handshake.IP, cpu.Handshake.Puerto, syscallINIT.Pid_proceso, syscallINIT.Pc, cpu.Handshake.Nombre)
}

func manejarDUMP_MEMORY(syscallDUMP globals.SyscallDump) {
	logSyscalls(syscallDUMP.PID, "DUMP_MEMORY")

	//log.Print("Se quiere loquear MapaProcesos en ManejarDUMP_MEMORY")
	globals.MapaProcesosMutex.Lock()
	proceso := globals.MapaProcesos[syscallDUMP.PID]
	globals.MapaProcesosMutex.Unlock()
	//log.Print("Se unloquea MapaProcesos en ManejarDUMP_MEMORY")

	utils_pm.EjecutarPlanificadorMedioPlazo(proceso, "Syscall Dump")
	general.LiberarCPU(syscallDUMP.NombreCPU)

	if general.EnviarDumpMemory(syscallDUMP.PID) {
		// Se desbloquea normalmente
		general.ActualizarPC(syscallDUMP.PID, syscallDUMP.PC)
		globals.MapaProcesosMutex.Lock()
		proceso := globals.MapaProcesos[syscallDUMP.PID]
		globals.MapaProcesosMutex.Unlock()
		estados.BlockedAReady(proceso)
	} else {
		general.FinalizarProceso(syscallDUMP.PID)
	}
}

func manejarEXIT(syscallEXIT globals.SyscallExit) {
	logSyscalls(syscallEXIT.PID, "EXIT")

	general.FinalizarProceso(syscallEXIT.PID)
	general.LiberarCPU(syscallEXIT.NombreCPU)

}

func logSyscalls(pid int64, syscall string) {
	slog.Info(fmt.Sprintf("## (%d) - Solicitó syscall: %s", pid, syscall))
}
