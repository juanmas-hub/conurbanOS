package estados

import (
	"log"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
)

// Se llama con estados mutex lockeado
func buscarProcesoEnBlocked(pid int64) int64 {

	colaBlocked := globals.ESTADOS.BLOCKED

	var posicion int64

	for indice, valor := range colaBlocked {
		if valor == pid {
			posicion = int64(indice)
			break
		}
	}

	return posicion
}

// Se llama con estados mutex lockeado
func buscarProcesoEnExecute(pid int64) int64 {
	colaExecute := globals.ESTADOS.EXECUTE

	var posicion int64

	for indice, valor := range colaExecute {
		if valor == pid {
			posicion = int64(indice)
			break
		}
	}

	return posicion
}

func buscarProcesoEnNew(pid int64) int64 {
	globals.EstadosMutex.Lock()
	log.Print("Se loqueo en buscarProcesoEnNew")
	colaNew := globals.ESTADOS.NEW
	globals.EstadosMutex.Unlock()
	log.Print("Se unloqueo en buscarProcesoEnNew")

	var posicion int64

	for indice, valor := range colaNew {
		if valor.Proceso.Pcb.Pid == pid {
			posicion = int64(indice)
			break
		}
	}

	return posicion
}

func buscarProcesoEnSuspBlocked(pid int64) int64 {
	globals.EstadosMutex.Lock()
	log.Print("Se loqueo en buscarProcesoEnSuspBlocked")
	colaSuspBlocked := globals.ESTADOS.SUSP_BLOCKED
	globals.EstadosMutex.Unlock()
	log.Print("Se unloqueo en buscarProcesoEnSuspBlocked")

	var posicion int64

	for indice, valor := range colaSuspBlocked {
		if valor == pid {
			posicion = int64(indice)
			break
		}
	}

	return posicion
}

func buscarProcesoEnSuspReady(pid int64) int64 {
	globals.EstadosMutex.Lock()
	log.Print("Se loqueo en buscarProcesoEnSuspReady")
	colaSuspReady := globals.ESTADOS.SUSP_READY
	globals.EstadosMutex.Unlock()
	log.Print("Se unloqueo en buscarProcesoEnSuspReady")

	var posicion int64

	for indice, valor := range colaSuspReady {
		if valor == pid {
			posicion = int64(indice)
			break
		}
	}

	return posicion
}

func buscarProcesoEnReady(pid int64) int64 {
	globals.EstadosMutex.Lock()
	log.Print("Se loqueo en buscarProcesoEnReady")
	colaReady := globals.ESTADOS.READY
	globals.EstadosMutex.Unlock()
	log.Print("Se unloqueo en buscarProcesoEnReady")

	var posicion int64

	for indice, valor := range colaReady {
		if valor == pid {
			posicion = int64(indice)
			break
		}
	}

	return posicion
}

// Busco la cola correspondiente y elimino el proceso
func EliminarProcesoDeSuCola(pid int64, estadoActual string) {
	switch estadoActual {
	case globals.BLOCKED:
		pos := buscarProcesoEnBlocked(pid)
		globals.ESTADOS.BLOCKED = append(globals.ESTADOS.BLOCKED[:pos], globals.ESTADOS.BLOCKED[pos+1:]...)
	case globals.EXECUTE:
		pos := buscarProcesoEnExecute(pid)
		globals.ESTADOS.EXECUTE = append(globals.ESTADOS.EXECUTE[:pos], globals.ESTADOS.EXECUTE[pos+1:]...)
	case globals.NEW:
		pos := buscarProcesoEnNew(pid)
		globals.ESTADOS.NEW = append(globals.ESTADOS.NEW[:pos], globals.ESTADOS.NEW[pos+1:]...)
	case globals.SUSP_BLOCKED:
		pos := buscarProcesoEnSuspBlocked(pid)
		globals.ESTADOS.SUSP_BLOCKED = append(globals.ESTADOS.SUSP_BLOCKED[:pos], globals.ESTADOS.SUSP_BLOCKED[pos+1:]...)
	case globals.SUSP_READY:
		pos := buscarProcesoEnSuspReady(pid)
		globals.ESTADOS.SUSP_READY = append(globals.ESTADOS.SUSP_READY[:pos], globals.ESTADOS.SUSP_READY[pos+1:]...)
	case globals.READY:
		pos := buscarProcesoEnReady(pid)
		globals.ESTADOS.READY = append(globals.ESTADOS.READY[:pos], globals.ESTADOS.READY[pos+1:]...)
	default:
		log.Printf("Error eliminando proceso PID: %d de su cola en EliminarDeSuCola", pid)
	}
}
