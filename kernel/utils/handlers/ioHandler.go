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

func FinalizacionIO(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var finalizacionIo globals.FinalizacionIO
	err := decoder.Decode(&finalizacionIo)
	if err != nil {
		slog.Debug(fmt.Sprintf("Error al decodificar mensaje: %s\n", err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	go manejarFinIO(finalizacionIo)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func manejarFinIO(finalizacionIo globals.FinalizacionIO) {

	globals.ListaIOsMutex.Lock()
	defer globals.ListaIOsMutex.Unlock()

	general.LogIntentoLockeo("MapaProcesos", "manejarFinIO")
	globals.MapaProcesosMutex.Lock()
	general.LogLockeo("MapaProcesos", "manejarFinIO")
	proceso, presente := globals.MapaProcesos[finalizacionIo.PID]
	if !presente {
		slog.Debug(fmt.Sprintf("El proceso %d que finalizo IO ya no existe en MapaProcesos. Probablemente finalizo", finalizacionIo.PID))
		globals.MapaProcesosMutex.Unlock()
		general.LogUnlockeo("MapaProcesos", "manejarFinIO")
		return
	}

	var nuevo_estado string

	general.LogIntentoLockeo("Estados", "manejarFinIO")
	globals.EstadosMutex.Lock()
	general.LogLockeo("Estados", "manejarFinIO")

	// Si esta en Susp Blocked lo paso a Susp Ready
	if proceso.Estado_Actual == globals.SUSP_BLOCKED {
		nuevo_estado = globals.SUSP_READY
		if !planificadores.CambiarEstado(proceso.Pcb.Pid, globals.SUSP_BLOCKED, globals.SUSP_READY) {
			slog.Debug(fmt.Sprintf("El proceso %d que finalizo IO no pudo cambiar de estado. Quizas cambio antes.", finalizacionIo.PID))
			globals.EstadosMutex.Unlock()
			general.LogUnlockeo("Estados", "manejarFinIO")
			globals.MapaProcesosMutex.Unlock()
			general.LogUnlockeo("MapaProcesos", "manejarFinIO")
			return
		}
	} else if proceso.Estado_Actual == globals.BLOCKED {
		// Si esta en Blocked lo paso Ready (no lo dice el enunciado!¡)
		nuevo_estado = globals.READY
		if !planificadores.CambiarEstado(proceso.Pcb.Pid, globals.BLOCKED, globals.READY) {
			slog.Debug(fmt.Sprintf("El proceso %d que finalizo IO no pudo cambiar de estado. Quizas cambio antes.", finalizacionIo.PID))
			globals.EstadosMutex.Unlock()
			general.LogUnlockeo("Estados", "manejarFinIO")
			globals.MapaProcesosMutex.Unlock()
			general.LogUnlockeo("MapaProcesos", "manejarFinIO")
			return
		}
		switch globals.KernelConfig.Scheduler_algorithm {
		case "FIFO", "SJF":
			general.Signal(globals.Sem_ProcesosEnReady)
		case "SRT":
			general.NotificarReplanifSRT()
		}
		slog.Debug(fmt.Sprintf("Notificando replanificación en manejarFinIO - Llego un proceso a READY"))
	}

	globals.EstadosMutex.Unlock()
	general.LogUnlockeo("Estados", "manejarFinIO")
	globals.MapaProcesosMutex.Unlock()
	general.LogUnlockeo("MapaProcesos", "manejarFinIO")

	// LOG : Fin de IO: ## (<PID>) finalizó IO y pasa a READY
	slog.Info(fmt.Sprintf("## (%d) finalizó IO y pasa a %s", finalizacionIo.PID, nuevo_estado))

	io := globals.MapaIOs[finalizacionIo.NombreIO]
	posInstanciaIo := BuscarInstanciaIO(finalizacionIo.NombreIO, finalizacionIo.NombreInstancia)

	if posInstanciaIo == -1 {
		slog.Debug(fmt.Sprintf("No se encontro la instancia de IO %s que tendria el PID: %d, probablemente porque se desconecto", finalizacionIo.NombreInstancia, finalizacionIo.PID))
		return
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
}

// Cuando se conecta una nueva IO, se fija si hay procesos esperando y los manda.
/*func IntentarPasarAIO(nombreIO string, instanciaIO globals.InstanciaIO) {

	globals.ListaIOsMutex.Lock()
	defer globals.ListaIOsMutex.Unlock()

	if len(globals.MapaIOs[nombreIO].ColaProcesosEsperando) > 0 {
		procesoAIO := globals.MapaIOs[nombreIO].ColaProcesosEsperando[0]
		instanciaIO.PidProcesoActual = procesoAIO.PID
		general.EnviarSolicitudIO(
			instanciaIO.Handshake.IP,
			instanciaIO.Handshake.Puerto,
			procesoAIO.PID,
			procesoAIO.Tiempo,
		)

		io := globals.MapaIOs[nombreIO]
		posInstanciaIo := BuscarInstanciaIO(nombreIO, instanciaIO.Handshake.NombreInstancia)
		io.ColaProcesosEsperando = io.ColaProcesosEsperando[1:]
		io.Instancias[posInstanciaIo] = instanciaIO
		globals.MapaIOs[nombreIO] = io
	}
}*/

func DesconexionIO(w http.ResponseWriter, r *http.Request) {
	// Cuando se desconecta un IO, se pasa a exit el proceso que estaba en el IO.

	decoder := json.NewDecoder(r.Body)
	var desconexionIO globals.DesconexionIO
	err := decoder.Decode(&desconexionIO)
	if err != nil {
		slog.Debug(fmt.Sprintf("Error al decodificar mensaje: %s\n", err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	globals.ListaIOsMutex.Lock()
	defer globals.ListaIOsMutex.Unlock()

	io := globals.MapaIOs[desconexionIO.NombreIO]

	pidProceso := desconexionIO.PID

	//slog.Debug(fmt.Sprintf("Se desconecto el IO: %s, que tenia el proceso de PID: %d", desconexionIO.NombreIO, pidProceso))

	// Saco la instancia de la cola de instancias
	posInstanciaIo := BuscarInstanciaIO(desconexionIO.NombreIO, desconexionIO.NombreInstancia)
	if posInstanciaIo < 0 {
		slog.Debug(fmt.Sprintf("Error buscando la instancia de IO de IP: %s, puerto: %d, que tendría el proceso: %d", desconexionIO.Ip, desconexionIO.Puerto, pidProceso))
	}
	io.Instancias = append(io.Instancias[:posInstanciaIo], io.Instancias[posInstanciaIo+1:]...)

	//slog.Debug(fmt.Sprint("Se saco la instancia de la cola de instancias: ", io.Instancias))

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

		// Elimino la entrada del mapa
		delete(globals.MapaIOs, desconexionIO.NombreIO)
	}
	globals.MapaIOs[desconexionIO.NombreIO] = io

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
	_, existeEntrada := globals.MapaIOs[nombreIO]
	if !existeEntrada {
		return false
	}
	if len(globals.MapaIOs[nombreIO].Instancias) <= 0 {
		return false
	}
	return true
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
