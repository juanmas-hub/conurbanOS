package utils_planifMedio

import (
	"time"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
	general "github.com/sisoputnfrba/tp-golang/kernel/utils/general"
	pl "github.com/sisoputnfrba/tp-golang/kernel/utils/planifLargo"
)

// ----- FUNCIONES EXPORTADAS -------

func ExecuteABlocked(proceso globals.Proceso) {
	// Esta funcion deberia llamarse cuando un proceso en ejecucion llama a IO con la syscall IO (desde syscallController)

	// -- Paso el proceso entre las colas
	// Como la cola de Execute 'no tiene' orden (todos los que estan en execute tienen una cpu ya ejecutando)
	// no se saca el primero de la cola como en las otras funciones
	proceso.Estado_Actual = globals.BLOCKED

	globals.MapaProcesosMutex.Lock()
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso
	globals.MapaProcesosMutex.Unlock()

	pos := general.BuscarProcesoEnExecute(proceso.Pcb.Pid)

	globals.EstadosMutex.Lock()
	globals.ESTADOS.EXECUTE = append(globals.ESTADOS.EXECUTE[:pos], globals.ESTADOS.EXECUTE[pos+1:]...)
	globals.ESTADOS.BLOCKED = append(globals.ESTADOS.BLOCKED, proceso.Pcb.Pid)
	globals.EstadosMutex.Unlock()

	// -- Timer hasta ser suspendido
	go timer(globals.KernelConfig.Suspension_time, proceso)

}

// ----- FUNCIONES LOCALES -------

func timer(tiempo int64, proceso globals.Proceso) {
	defer sigueBloqueado(proceso)
	duracion := time.Duration(tiempo) * time.Millisecond
	time.Sleep(duracion)
}

func sigueBloqueado(proceso globals.Proceso) {
	// Si sigue bloqueado (en IO) hay que suspenderlo
	// Para que no siga bloqueado, el proceso tuvo que terminar su IO (lo recibimos como mensaje desde IO, siendo kernel servidor)
	// Cuando kernel reciba de IO el mensaje, ahí le cambiamos el estado

	globals.MapaProcesosMutex.Lock()
	procesoActualmente := globals.MapaProcesos[proceso.Pcb.Pid]
	globals.MapaProcesosMutex.Unlock()

	if procesoActualmente.Estado_Actual == globals.BLOCKED {
		blockedASuspBlocked(proceso)

		// Aca hay q hacer un par de cosas mas pero me tengo q ir
	}
}

func blockedASuspBlocked(proceso globals.Proceso) {
	// Aviso a memoria para swappear (hay q hacerlo)

	// Muevo el proceso en la colas
	proceso.Estado_Actual = globals.SUSP_BLOCKED

	globals.MapaProcesosMutex.Lock()
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso
	globals.MapaProcesosMutex.Unlock()

	pos := general.BuscarProcesoEnBlocked(proceso.Pcb.Pid)

	globals.EstadosMutex.Lock()
	globals.ESTADOS.BLOCKED = append(globals.ESTADOS.BLOCKED[:pos], globals.ESTADOS.BLOCKED[pos+1:]...)
	globals.ESTADOS.SUSP_BLOCKED = append(globals.ESTADOS.SUSP_BLOCKED, proceso.Pcb.Pid)
	globals.EstadosMutex.Unlock()

	// Libere memoria => llamo a nuevos procesos
	pl.PasarProcesosAReady()
}
