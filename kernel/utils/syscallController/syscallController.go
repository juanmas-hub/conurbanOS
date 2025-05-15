package utils_syscallController

import (
	"encoding/json"
	"log"
	"net/http"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
	utils_general "github.com/sisoputnfrba/tp-golang/kernel/utils/general"
	utils_pm "github.com/sisoputnfrba/tp-golang/kernel/utils/planifMedio"
)

// Cuando la CPU detecta una syscall, nos envía a kernel y nosotros la manejamos:

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
		posIo, existe := utils_general.ObtenerIO(syscallIO.Nombre)

		if !existe {
			utils_general.ProcesoAExit(syscallIO.PID)
		} else {

			// Bloqueo el proceso
			globals.MapaProcesosMutex.Lock()
			proceso := globals.MapaProcesos[syscallIO.PID]
			globals.MapaProcesosMutex.Unlock()

			utils_pm.BloquearProcesoDesdeExecute(proceso)

			// Si esta libre, envio solicitud, sino agrego a la cola
			io := globals.ListaIOs[posIo]
			if len(io.ColaProcesosEsperando) == 0 {
				globals.ListaIOs[posIo].PidProcesoActual = syscallIO.PID
				utils_general.EnviarSolicitudIO(io.Handshake.IP, io.Handshake.Puerto, syscallIO.PID, syscallIO.Tiempo)
			}
			io.ColaProcesosEsperando = append(io.ColaProcesosEsperando, syscallIO)
			globals.ListaIOs[posIo] = io
		}

		globals.ListaIOsMutex.Unlock()
	}()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func ManejarINIT_PROC(w http.ResponseWriter, r *http.Request) {

}

func ManejarDUMP_MEMORY(w http.ResponseWriter, r *http.Request) {

}

func ManejarEXIT(w http.ResponseWriter, r *http.Request) {

}
