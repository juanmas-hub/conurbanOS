package utils_syscallController

import (
	"encoding/json"
	"log"
	"net/http"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
	general "github.com/sisoputnfrba/tp-golang/kernel/utils/general"
	utils_pm "github.com/sisoputnfrba/tp-golang/kernel/utils/planifMedio"
)

// Cuando la CPU detecta una syscall, nos envía a kernel y nosotros la manejamos:
// En cada syscall hay que actualizarle el PC a los procesos!!

func ManejarIO(w http.ResponseWriter, r *http.Request) {
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

	go func() {
		globals.ListaIOsMutex.Lock()
		posIo, existe := general.ObtenerIO(syscallIO.NombreIO)

		if !existe {
			general.ProcesoAExit(syscallIO.PID)
		} else {

			// Bloqueo el proceso y le actualizo el PC
			globals.MapaProcesosMutex.Lock()
			proceso := globals.MapaProcesos[syscallIO.PID]
			proceso.Pcb.PC = syscallIO.PC
			globals.MapaProcesosMutex.Unlock()

			utils_pm.BloquearProcesoDesdeExecute(proceso)

			// Si esta libre, envio solicitud, sino agrego a la cola
			io := globals.ListaIOs[posIo]
			if len(io.ColaProcesosEsperando) == 0 {
				globals.ListaIOs[posIo].PidProcesoActual = syscallIO.PID
				general.EnviarSolicitudIO(io.Handshake.IP, io.Handshake.Puerto, syscallIO.PID, syscallIO.Tiempo)
			}
			io.ColaProcesosEsperando = append(io.ColaProcesosEsperando, syscallIO)
			globals.ListaIOs[posIo] = io
		}

		globals.ListaIOsMutex.Unlock()

		// Libero CPU
		posCpu := general.BuscarCpu(syscallIO.NombreCPU)
		globals.ListaCPUsMutex.Lock()
		globals.ListaCPUs[posCpu].EstaLibre = true
		globals.ListaCPUsMutex.Unlock()
		general.Signal(globals.Sem_Cpus)
	}()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func ManejarINIT_PROC(w http.ResponseWriter, r *http.Request) {

}

func ManejarDUMP_MEMORY(w http.ResponseWriter, r *http.Request) {

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
		general.FinalizarProceso(syscallEXIT.PID, syscallEXIT.NombreCPU)

	}()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
