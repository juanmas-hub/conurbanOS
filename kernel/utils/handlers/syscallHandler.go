package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
	general "github.com/sisoputnfrba/tp-golang/kernel/utils/general"
	planificadores "github.com/sisoputnfrba/tp-golang/kernel/utils/planificadores"
)

// Cuando la CPU detecta una syscall, nos envía a kernel y nosotros la manejamos:
// En todas las syscalls la CPU "se libera" y queda esperando para simular el tiempo que ejecuta el SO
// - En IO el proceso se bloquea, entonces directamente el planificador de corto plazo replanifica.
// - En INIT PROC la CPU no la indicamos como "libre" porque tiene que volver a ejecutar el mismo proceso

func RecibirIO(w http.ResponseWriter, r *http.Request) {
	// Recibo desde CPU la syscall IO y le envío solicitud a la IO correspondiente

	decoder := json.NewDecoder(r.Body)
	var syscallIO globals.SyscallIO
	err := decoder.Decode(&syscallIO)
	if err != nil {
		slog.Debug(fmt.Sprintf("Error al decodificar syscallIO: %s\n", err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar syscallIO"))
		return
	}

	go manejarIO(syscallIO)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func RecibirINIT_PROC(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var syscallINIT globals.SyscallInit
	err := decoder.Decode(&syscallINIT)
	if err != nil {
		slog.Debug(fmt.Sprintf("Error al decodificar SyscallInit: %s\n", err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar syscallINIT"))
		return
	}

	go manejarInit_Proc(syscallINIT)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func RecibirDUMP_MEMORY(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var syscallDUMP globals.SyscallDump
	err := decoder.Decode(&syscallDUMP)
	if err != nil {
		slog.Debug(fmt.Sprintf("Error al decodificar SyscallDump: %s\n", err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar SyscallDump"))
		return
	}

	go manejarDUMP_MEMORY(syscallDUMP)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func RecibirEXIT(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var syscallEXIT globals.SyscallExit
	err := decoder.Decode(&syscallEXIT)
	if err != nil {
		slog.Debug(fmt.Sprintf("Error al decodificar SyscallExit: %s\n", err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar SyscallExit"))
		return
	}

	go manejarEXIT(syscallEXIT)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// ---- LOGICA

func manejarIO(syscallIO globals.SyscallIO) {

	globals.ListaIOsMutex.Lock()
	globals.MapaProcesosMutex.Lock()

	proceso, existe := globals.MapaProcesos[syscallIO.PID]
	if !existe {
		slog.Debug(fmt.Sprintf("No se atiende IO porque PID %d no existe. Posiblemente finalizo", syscallIO.PID))
		globals.MapaProcesosMutex.Unlock()
		globals.ListaIOsMutex.Unlock()
		return
	}
	if proceso.Estado_Actual != globals.EXECUTE {
		slog.Debug(fmt.Sprintf("No se atiende IO porque PID %d no esta en EXECUTE. Posiblemente se interrumpio", syscallIO.PID))
		globals.MapaProcesosMutex.Unlock()
		globals.ListaIOsMutex.Unlock()
		return
	}

	// Motivo de Bloqueo: ## (<PID>) - Bloqueado por IO: <DISPOSITIVO_IO>
	slog.Info(fmt.Sprintf("## (%d) - Bloqueado por IO: %s", syscallIO.PID, syscallIO.NombreIO))

	if !VerificarExistenciaIO(syscallIO.NombreIO) {
		slog.Debug(fmt.Sprint("No existe IO: ", syscallIO.NombreIO))
		globals.MapaProcesosMutex.Unlock()
		globals.ListaIOsMutex.Unlock()
		planificadores.FinalizarProceso(syscallIO.PID, globals.EXECUTE)
		general.LiberarCPU(syscallIO.NombreCPU)
		return
	}

	logSyscalls(syscallIO.PID, "IO")

	io := globals.MapaIOs[syscallIO.NombreIO]
	general.ActualizarPC(syscallIO.PID, syscallIO.PC)

	globals.MapaProcesosMutex.Unlock()
	general.LogUnlockeo("MapaProcesos", "manejarIO")

	// Si hay instancias libres, envio solicitud, sino agrego a la cola
	instanciaIo, pos, hayLibre := BuscarInstanciaIOLibre(syscallIO.NombreIO)

	if hayLibre {
		slog.Debug(fmt.Sprint("Seleccionada IO libre: ", instanciaIo))
		instanciaIo.PidProcesoActual = syscallIO.PID
		general.EnviarSolicitudIO(instanciaIo.Handshake.IP, instanciaIo.Handshake.Puerto, syscallIO.PID, syscallIO.Tiempo)
		io.Instancias[pos] = instanciaIo
	} else {
		io.ColaProcesosEsperando = append(io.ColaProcesosEsperando, syscallIO)
	}

	slog.Debug(fmt.Sprintf("Tiempo de IO de PID (%d): %d", syscallIO.PID, syscallIO.Tiempo))
	globals.MapaIOs[syscallIO.NombreIO] = io
	globals.ListaIOsMutex.Unlock()

	general.LogIntentoLockeo("MapaProcesos", "manejarIO")
	globals.MapaProcesosMutex.Lock()
	general.LogLockeo("MapaProcesos", "manejarIO")
	proceso, presente := globals.MapaProcesos[syscallIO.PID]
	globals.MapaProcesosMutex.Unlock()
	general.LogUnlockeo("MapaProcesos", "manejarIO")
	if presente {
		planificadores.EjecutarPlanificadorMedioPlazo(proceso, "Syscall IO")
	} else {
		slog.Debug(fmt.Sprintf("El proceso %d que solicito IO ya no existe en MapaProcesos. Probablemente finalizo", syscallIO.PID))
	}
	general.LiberarCPU(syscallIO.NombreCPU)

}

func manejarInit_Proc(syscallINIT globals.SyscallInit) {
	logSyscalls(syscallINIT.Pid_proceso, "INIT_PROC")
	planificadores.CrearProcesoNuevo(syscallINIT.Archivo, syscallINIT.Tamanio)

	// El proceso vuelve a ejecutar
	globals.ListaCPUsMutex.Lock()
	posCpu := general.BuscarCpu(syscallINIT.Nombre_CPU)
	cpu := globals.ListaCPUs[posCpu]
	globals.ListaCPUsMutex.Unlock()
	general.EnviarProcesoAEjecutar_ACPU(cpu.Handshake.IP, cpu.Handshake.Puerto, syscallINIT.Pid_proceso, syscallINIT.Pc, cpu.Handshake.Nombre)
}

func manejarDUMP_MEMORY(syscallDUMP globals.SyscallDump) {
	logSyscalls(syscallDUMP.PID, "DUMP_MEMORY")

	general.LogIntentoLockeo("MapaProcesos", "manejarDUMP_MEMORY")
	globals.MapaProcesosMutex.Lock()
	general.LogLockeo("MapaProcesos", "manejarDUMP_MEMORY")
	proceso, presente := globals.MapaProcesos[syscallDUMP.PID]
	if !presente {
		slog.Debug(fmt.Sprintf("El proceso %d que solicito DUMP ya no existe en MapaProcesos. Probablemente finalizo", syscallDUMP.PID))
		return
	}
	globals.MapaProcesosMutex.Unlock()
	general.LogUnlockeo("MapaProcesos", "manejarDUMP_MEMORY")

	// Si se pudo bloquear
	if planificadores.EjecutarPlanificadorMedioPlazo(proceso, "Syscall Dump") {

		// Si se pudo dumpear
		if general.EnviarDumpMemory(syscallDUMP.PID) {

			general.LogIntentoLockeo("MapaProcesos", "manejarDUMP_MEMORY")
			globals.MapaProcesosMutex.Lock()
			general.LogLockeo("MapaProcesos", "manejarDUMP_MEMORY")
			proceso, presente = globals.MapaProcesos[syscallDUMP.PID]
			if !presente {
				slog.Debug(fmt.Sprintf("El proceso %d que acaba de DUMPEAR ya no existe en MapaProcesos. Probablemente finalizo", syscallDUMP.PID))
			} else {
				general.ActualizarPC(syscallDUMP.PID, syscallDUMP.PC)

				general.LogIntentoLockeo("Estados", "manejarDUMP_MEMORY")
				globals.EstadosMutex.Lock()
				general.LogLockeo("Estados", "manejarDUMP_MEMORY")
				ok := planificadores.CambiarEstado(proceso.Pcb.Pid, globals.BLOCKED, globals.READY)
				if !ok {
					slog.Debug(fmt.Sprintf("El proceso %d que acaba de DUMPEAR no pudo cambiar de estado.", syscallDUMP.PID))
					globals.EstadosMutex.Unlock()
					globals.MapaProcesosMutex.Unlock()
					general.LogUnlockeo("MapaProcesos", "manejarDUMP_MEMORY")
					return
				}
				globals.EstadosMutex.Unlock()
				slog.Debug(fmt.Sprint("Se llego hasta unlockear Estados Mutex"))
				switch globals.KernelConfig.Scheduler_algorithm {
				case "FIFO", "SJF":
					slog.Debug(fmt.Sprintf("Notificando replanificación en manejarDUMP_MEMORY - Nuevo proceso en ready"))
					general.Signal(globals.Sem_ProcesosEnReady)
				case "SRT":
					slog.Debug(fmt.Sprintf("Notificando replanificación en manejarDUMP_MEMORY - Nuevo proceso en ready"))
					general.NotificarReplanifSRT()
				}
			}
			globals.MapaProcesosMutex.Unlock()
			general.LogUnlockeo("MapaProcesos", "manejarDUMP_MEMORY")

		} else {
			// Si no se pudo dumpear
			planificadores.FinalizarProceso(syscallDUMP.PID, globals.BLOCKED)
		}
	}

	general.LiberarCPU(syscallDUMP.NombreCPU)
}

func manejarEXIT(syscallEXIT globals.SyscallExit) {

	logSyscalls(syscallEXIT.PID, "EXIT")

	planificadores.FinalizarProceso(syscallEXIT.PID, globals.EXECUTE)
	general.LiberarCPU(syscallEXIT.NombreCPU)

}

func logSyscalls(pid int64, syscall string) {
	slog.Info(fmt.Sprintf("## (%d) - Solicitó syscall: %s", pid, syscall))
}
