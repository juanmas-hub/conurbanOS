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

	go func() {
		manejarFinIO(finalizacionIo)
	}()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func manejarFinIO(finalizacionIo globals.FinalizacionIO) {

	globals.ListaIOsMutex.Lock()
	io := globals.MapaIOs[finalizacionIo.NombreIO]
	posInstanciaIo := BuscarInstanciaIO(finalizacionIo.NombreIO, finalizacionIo.NombreInstancia)
	log.Print(globals.MapaIOs)

	if posInstanciaIo == -1 {
		log.Printf("Error buscando instancia de IO de nombre: %s, con el proceso: %d", finalizacionIo.NombreIO, finalizacionIo.PID)
	}
	instanciaIo := io.Instancias[posInstanciaIo]

	// Cambio el PID del proceso actual
	instanciaIo.PidProcesoActual = -1
	io.Instancias[posInstanciaIo] = instanciaIo

	globals.MapaIOs[finalizacionIo.NombreIO] = io

	//log.Print("Se quiere loquear MapaProcesos en manejarFinIO")
	globals.MapaProcesosMutex.Lock()
	proceso := globals.MapaProcesos[finalizacionIo.PID]
	globals.MapaProcesosMutex.Unlock()
	//log.Print("Se unloquea MapaProcesos en manejarFinIO")

	var nuevo_estado string

	// Si esta en Susp Blocked lo paso a Susp Ready
	if proceso.Estado_Actual == globals.SUSP_BLOCKED {
		nuevo_estado = globals.SUSP_READY
		planificadores.SuspBlockedASuspReady(proceso)
	}

	// Si esta en Blocked lo paso Ready (no lo dice el enunciado!¡)
	if proceso.Estado_Actual == globals.BLOCKED {
		nuevo_estado = globals.READY
		planificadores.BlockedAReady(proceso)
	}

	// LOG : Fin de IO: ## (<PID>) finalizó IO y pasa a READY
	slog.Info(fmt.Sprintf("## (%d) finalizó IO y pasa a %s", finalizacionIo.PID, nuevo_estado))

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

}

func DesconexionIO(w http.ResponseWriter, r *http.Request) {
	// Cuando se desconecta un IO, se pasa a exit el proceso que estaba en el IO.

	decoder := json.NewDecoder(r.Body)
	var desconexionIO globals.DesconexionIO
	err := decoder.Decode(&desconexionIO)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	globals.ListaIOsMutex.Lock()
	io := globals.MapaIOs[desconexionIO.NombreIO]
	globals.ListaIOsMutex.Unlock()

	pidProceso := desconexionIO.PID

	slog.Debug(fmt.Sprintf("Se desconecto el IO: %s, que tenia el proceso de PID: %d", desconexionIO.NombreIO, pidProceso))

	// Saco la instancia de la cola de instancias
	globals.ListaIOsMutex.Lock()
	posInstanciaIo := BuscarInstanciaIO(desconexionIO.NombreIO, desconexionIO.NombreInstancia)
	if posInstanciaIo == -2 {
		log.Printf("Error buscando la instancia de IO de IP: %s, puerto: %d, que tendría el proceso: %d", desconexionIO.Ip, desconexionIO.Puerto, pidProceso)
	}
	io.Instancias = append(io.Instancias[:posInstanciaIo], io.Instancias[posInstanciaIo+1:]...)
	globals.ListaIOsMutex.Unlock()

	// Si habia proceso ejecutando
	if pidProceso != -1 {

		// Finalizo proceso que esta ejecuando en esa IO
		planificadores.FinalizarProceso(pidProceso)
	}

	// Si no quedan mas instancias
	if len(io.Instancias) == 0 {
		// Finalizo todos los procesos de la cola esperando esa IO
		for i := range io.ColaProcesosEsperando {
			planificadores.FinalizarProceso(io.ColaProcesosEsperando[i].PID)
		}
	}

	globals.ListaIOsMutex.Lock()
	globals.MapaIOs[desconexionIO.NombreIO] = io
	globals.ListaIOsMutex.Unlock()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// Dada un nombre de IO, busca una instancia libre. Devuelve la instancia, la posicion en la cola de instancias y si hay instancia libre. Se llama con Lista IO muteada
func BuscarInstanciaIOLibre(nombreIO string) (globals.InstanciaIO, int, bool) {
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
func VerificarExistenciaIO(nombreIO string) bool {
	_, existe := globals.MapaIOs[nombreIO]
	return existe
}

// Dado un nombre IO e instancia, retorna posicion en cola de instancias. Se llama con Lista IO muteada
func BuscarInstanciaIO(nombreIO string, nombreInstancia string) int {

	io := globals.MapaIOs[nombreIO]

	for i := range io.Instancias {
		if io.Instancias[i].Handshake.NombreInstancia == nombreInstancia {
			return i
		}
	}

	return -1
}
