package utils_syscallController

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
	general "github.com/sisoputnfrba/tp-golang/kernel/utils/general"
	utils_pl "github.com/sisoputnfrba/tp-golang/kernel/utils/planifLargo"
	utils_pm "github.com/sisoputnfrba/tp-golang/kernel/utils/planifMedio"
)

// Cuando la CPU detecta una syscall, nos envía a kernel y nosotros la manejamos:
// En cada syscall hay que actualizarle el PC a los procesos!!

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

	log.Println("Hubo una Syscall IO")
	log.Printf("%+v\n", syscallIO)

	go ManejarIO(syscallIO)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func ManejarIO(syscallIO globals.SyscallIO) {
	globals.ListaIOsMutex.Lock()
	existe := verificarExistenciaIO(syscallIO.NombreIO)
	nombreIO := syscallIO.NombreIO

	log.Print("Vino una syscall IO a ManejarIO:", syscallIO)

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
		instanciaIo, pos, hayLibre := buscarInstanciaIOLibre(syscallIO.NombreIO)
		if hayLibre {
			log.Print("Seleccionada IO libre: ", instanciaIo)
			instanciaIo.PidProcesoActual = syscallIO.PID
			general.EnviarSolicitudIO(instanciaIo.Handshake.IP, instanciaIo.Handshake.Puerto, syscallIO.PID, syscallIO.Tiempo)
			io.Instancias[pos] = instanciaIo
		} else {
			io.ColaProcesosEsperando = append(io.ColaProcesosEsperando, syscallIO)
		}
		utils_pm.BloquearProcesoDesdeExecute(proceso, "Syscall IO")
		globals.MapaIOs[nombreIO] = io

	}

	globals.ListaIOsMutex.Unlock()
	// Motivo de Bloqueo: ## (<PID>) - Bloqueado por IO: <DISPOSITIVO_IO>
	slog.Info(fmt.Sprintf("## (%d) - Bloqueado por IO: %s", syscallIO.PID, syscallIO.NombreIO))

	logSyscalls(syscallIO.PID, "IO")

}

func ManejarINIT_PROC(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var syscallINIT globals.SyscallInit
	err := decoder.Decode(&syscallINIT)
	if err != nil {
		log.Printf("Error al decodificar SyscallInit: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar syscallINIT"))
		return
	}

	log.Println("Hubo una syscallINIT")
	log.Printf("%+v\n", syscallINIT)

	go func() {

		go utils_pl.CrearProcesoNuevo(syscallINIT.Archivo, syscallINIT.Tamanio)

		// El proceso vuelve a ejecutar
		globals.ListaCPUsMutex.Lock()
		posCpu := general.BuscarCpu(syscallINIT.Nombre_CPU)
		cpu := globals.ListaCPUs[posCpu]
		globals.ListaCPUsMutex.Unlock()
		general.EnviarProcesoAEjecutar_ACPU(cpu.Handshake.IP, cpu.Handshake.Puerto, syscallINIT.Pid_proceso, syscallINIT.Pc)

		logSyscalls(syscallINIT.Pid_proceso, "INIT_PROC")
	}()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func ManejarDUMP_MEMORY(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var syscallDUMP globals.SyscallDump
	err := decoder.Decode(&syscallDUMP)
	if err != nil {
		log.Printf("Error al decodificar SyscallDump: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar SyscallDump"))
		return
	}

	log.Println("Hubo una SyscallDump")
	log.Printf("%+v\n", syscallDUMP)

	go func() {

		//log.Print("Se quiere loquear MapaProcesos en ManejarDUMP_MEMORY")
		globals.MapaProcesosMutex.Lock()
		proceso := globals.MapaProcesos[syscallDUMP.PID]
		globals.MapaProcesosMutex.Unlock()
		//log.Print("Se unloquea MapaProcesos en ManejarDUMP_MEMORY")

		utils_pm.BloquearProcesoDesdeExecute(proceso, "Syscall Dump")
		general.LiberarCPU(syscallDUMP.NombreCPU)

		if general.EnviarDumpMemory(syscallDUMP.PID) {
			// Se desbloquea normalmente
			general.BlockedAReady(proceso)
		} else {
			general.FinalizarProceso(syscallDUMP.PID)
		}

		logSyscalls(syscallDUMP.PID, "DUMP_MEMORY")
	}()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func ManejarEXIT(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var syscallEXIT globals.SyscallExit
	err := decoder.Decode(&syscallEXIT)
	if err != nil {
		log.Printf("Error al decodificar SyscallExit: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar SyscallExit"))
		return
	}

	log.Println("Hubo una Syscall EXIT")
	log.Printf("%+v\n", syscallEXIT)

	go func() {
		general.FinalizarProceso(syscallEXIT.PID)
		general.LiberarCPU(syscallEXIT.NombreCPU)

		logSyscalls(syscallEXIT.PID, "EXIT")
	}()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// ------ FUNCIONES LOCALES --------

// Dada un nombre de IO, busca una instancia libre. Devuelve la instancia, la posicion en la cola de instancias y si hay instancia libre. Se llama con Lista IO muteada
func buscarInstanciaIOLibre(nombreIO string) (globals.InstanciaIO, int, bool) {
	var instancia globals.InstanciaIO

	io := globals.MapaIOs[nombreIO]

	for i := range io.Instancias {
		if io.Instancias[i].PidProcesoActual == -1 {
			// Esta libre
			return io.Instancias[i], i, true
		}
	}

	return instancia, -1, false
}

// Dado un nombre de IO, devuelve si existe. Se llama con Lista IO muteada.
func verificarExistenciaIO(nombreIO string) bool {
	_, existe := globals.MapaIOs[nombreIO]
	return existe
}

func logSyscalls(pid int64, syscall string) {
	slog.Info(fmt.Sprintf("## (%d) - Solicitó syscall: %s", pid, syscall))
}
