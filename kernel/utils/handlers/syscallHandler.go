package handlers

import (
	"encoding/json"
	"fmt"
	"log"
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
		log.Printf("Error al decodificar syscallIO: %s\n", err.Error())
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
		log.Printf("Error al decodificar SyscallInit: %s\n", err.Error())
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
		log.Printf("Error al decodificar SyscallDump: %s\n", err.Error())
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
		log.Printf("Error al decodificar SyscallExit: %s\n", err.Error())
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
	existe := VerificarExistenciaIO(syscallIO.NombreIO)
	nombreIO := syscallIO.NombreIO

	logSyscalls(syscallIO.PID, "IO")

	// Motivo de Bloqueo: ## (<PID>) - Bloqueado por IO: <DISPOSITIVO_IO>
	slog.Info(fmt.Sprintf("## (%d) - Bloqueado por IO: %s", syscallIO.PID, syscallIO.NombreIO))
	if !existe {
		globals.ListaIOsMutex.Unlock()
		planificadores.FinalizarProceso(syscallIO.PID)
	} else {

		// Bloqueo el proceso y le actualizo el PC
		general.LogIntentoLockeo("Mapa Procesos", "manejarIO")
		globals.MapaProcesosMutex.Lock()
		general.LogLockeo("Mapa Procesos", "manejarIO")
		general.ActualizarPC(syscallIO.PID, syscallIO.PC)
		proceso := globals.MapaProcesos[syscallIO.PID]
		globals.MapaProcesosMutex.Unlock()
		general.LogUnlockeo("Mapa Procesos", "manejarIO")

		// Si hay instancias libres, envio solicitud, sino agrego a la cola
		io := globals.MapaIOs[nombreIO]
		instanciaIo, pos, hayLibre := BuscarInstanciaIOLibre(syscallIO.NombreIO)
		if hayLibre {
			log.Print("Seleccionada IO libre: ", instanciaIo)
			instanciaIo.PidProcesoActual = syscallIO.PID
			general.EnviarSolicitudIO(instanciaIo.Handshake.IP, instanciaIo.Handshake.Puerto, syscallIO.PID, syscallIO.Tiempo)
			io.Instancias[pos] = instanciaIo
		} else {
			io.ColaProcesosEsperando = append(io.ColaProcesosEsperando, syscallIO)
		}
		slog.Debug(fmt.Sprintf("Tiempo de IO de PID (%d): %d", syscallIO.PID, syscallIO.Tiempo))
		globals.MapaIOs[nombreIO] = io
		globals.ListaIOsMutex.Unlock()
		planificadores.EjecutarPlanificadorMedioPlazo(proceso, "Syscall IO")

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

	general.LogIntentoLockeo("Mapa Procesos", "manejarDUMP_MEMORY")
	globals.MapaProcesosMutex.Lock()
	general.LogLockeo("Mapa Procesos", "manejarDUMP_MEMORY")
	proceso := globals.MapaProcesos[syscallDUMP.PID]
	globals.MapaProcesosMutex.Unlock()
	general.LogUnlockeo("Mapa Procesos", "manejarDUMP_MEMORY")

	planificadores.EjecutarPlanificadorMedioPlazo(proceso, "Syscall Dump")
	general.LiberarCPU(syscallDUMP.NombreCPU)

	if general.EnviarDumpMemory(syscallDUMP.PID) {
		// Se desbloquea normalmente
		general.LogIntentoLockeo("Mapa Procesos", "manejarDUMP_MEMORY 2")
		globals.MapaProcesosMutex.Lock()
		general.LogLockeo("Mapa Procesos", "manejarDUMP_MEMORY 2")
		general.ActualizarPC(syscallDUMP.PID, syscallDUMP.PC)
		proceso := globals.MapaProcesos[syscallDUMP.PID]
		globals.MapaProcesosMutex.Unlock()
		general.LogUnlockeo("Mapa Procesos", "manejarDUMP_MEMORY 2")
		planificadores.BlockedAReady(proceso)
	} else {
		planificadores.FinalizarProceso(syscallDUMP.PID)
	}
}

func manejarEXIT(syscallEXIT globals.SyscallExit) {
	logSyscalls(syscallEXIT.PID, "EXIT")

	//general.FinalizarProceso(syscallEXIT.PID)
	planificadores.FinalizarProceso(syscallEXIT.PID)
	general.LiberarCPU(syscallEXIT.NombreCPU)

}

func logSyscalls(pid int64, syscall string) {
	slog.Info(fmt.Sprintf("## (%d) - Solicitó syscall: %s", pid, syscall))
}
