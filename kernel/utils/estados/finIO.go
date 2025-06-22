package estados

// No tiene sentido que este aca pero toca

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
	general "github.com/sisoputnfrba/tp-golang/kernel/utils/general"
)

func FinalizacionIO(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var finalizacionIo globals.FinalizacionIO
	err := decoder.Decode(&finalizacionIo)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	log.Printf("Finalizo el IO del PID: %d", finalizacionIo.PID)
	log.Printf("%+v\n", finalizacionIo)

	go func() {
		manejarFinIO(finalizacionIo)
	}()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func manejarFinIO(finalizacionIo globals.FinalizacionIO) {
	globals.ListaIOsMutex.Lock()

	io := globals.MapaIOs[finalizacionIo.NombreIO]
	posInstanciaIo := general.BuscarInstanciaIO(finalizacionIo.NombreIO, finalizacionIo.PID)
	if posInstanciaIo == -1 {
		log.Printf("Error buscando instancia de IO de nombre: %s, con el proceso: %d", finalizacionIo.NombreIO, finalizacionIo.PID)
	}
	instanciaIo := io.Instancias[posInstanciaIo]

	// Cambio el PID del proceso actual
	instanciaIo.PidProcesoActual = -1
	io.Instancias[posInstanciaIo] = instanciaIo

	globals.MapaIOs[finalizacionIo.NombreIO] = io

	// Si hay procesos esperando IO, envio solicitud
	if len(globals.MapaIOs[finalizacionIo.NombreIO].ColaProcesosEsperando) > 0 {
		procesoAIO := globals.MapaIOs[finalizacionIo.NombreIO].ColaProcesosEsperando[0]
		instanciaIo.PidProcesoActual = procesoAIO.PID
		general.EnviarSolicitudIO(
			instanciaIo.Handshake.IP,
			instanciaIo.Handshake.Puerto,
			procesoAIO.PID,
			procesoAIO.Tiempo,
		)

		// Saco al nuevo proceso de la cola de procesos esperando
		io.ColaProcesosEsperando = io.ColaProcesosEsperando[1:]
	}

	io.Instancias[posInstanciaIo] = instanciaIo
	globals.MapaIOs[finalizacionIo.NombreIO] = io

	globals.ListaIOsMutex.Unlock()

	//log.Print("Se quiere loquear MapaProcesos en manejarFinIO")
	globals.MapaProcesosMutex.Lock()
	proceso := globals.MapaProcesos[finalizacionIo.PID]
	globals.MapaProcesosMutex.Unlock()
	//log.Print("Se unloquea MapaProcesos en manejarFinIO")

	var nuevo_estado string

	// Si esta en Susp Blocked lo paso a Susp Ready
	if proceso.Estado_Actual == globals.SUSP_BLOCKED {
		nuevo_estado = globals.SUSP_READY
		SuspBlockedASuspReady(proceso)
	}

	// Si esta en Blocked lo paso Ready (no lo dice el enunciado!¡)
	if proceso.Estado_Actual == globals.BLOCKED {
		nuevo_estado = globals.READY
		BlockedAReady(proceso)
	}

	// LOG : Fin de IO: ## (<PID>) finalizó IO y pasa a READY
	slog.Info(fmt.Sprintf("## (%d) finalizó IO y pasa a %s", finalizacionIo.PID, nuevo_estado))
}
