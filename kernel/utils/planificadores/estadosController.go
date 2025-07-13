package planificadores

import (
	"fmt"
	"log"
	"log/slog"
	"time"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
	general "github.com/sisoputnfrba/tp-golang/kernel/utils/general"
)

func SuspBlockedASuspReady(proceso globals.Proceso) {
	proceso = general.ActualizarMetricas(proceso, proceso.Estado_Actual)
	proceso.Estado_Actual = globals.SUSP_READY
	globals.MapaProcesosMutex.Lock()
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso
	globals.MapaProcesosMutex.Unlock()

	pos := buscarProcesoEnSuspBlocked(proceso.Pcb.Pid)

	slog.Debug("Se quiere lockear en SuspBlockedASuspReady")
	globals.EstadosMutex.Lock()
	slog.Debug("Se lockear en SuspBlockedASuspReady")
	globals.ESTADOS.SUSP_BLOCKED = append(globals.ESTADOS.SUSP_BLOCKED[:pos], globals.ESTADOS.SUSP_BLOCKED[pos+1:]...)
	globals.ESTADOS.SUSP_READY = append(globals.ESTADOS.SUSP_READY, proceso.Pcb.Pid)
	slog.Debug("Se quiere deslockear en SuspBlockedASuspReady")
	globals.EstadosMutex.Unlock()
	slog.Debug("Se deslockear en SuspBlockedASuspReady")

	globals.DeDondeSeLlamaMutex.Lock()
	globals.DeDondeSeLlamaPasarProcesosAReady = "SUSP READY"
	globals.DeDondeSeLlamaMutex.Unlock()
	globals.SignalPasarProcesoAReady()

	// LOG Cambio de Estado: ## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado SUSP_BLOCKED al estado SUSP_READY", proceso.Pcb.Pid))
}

func BlockedAReady(proceso globals.Proceso) {
	proceso = general.ActualizarMetricas(proceso, proceso.Estado_Actual)
	proceso.Estado_Actual = globals.READY
	globals.MapaProcesosMutex.Lock()
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso
	globals.MapaProcesosMutex.Unlock()

	globals.EstadosMutex.Lock()
	pos := buscarProcesoEnBlocked(proceso.Pcb.Pid)

	slog.Debug("Se quiere lockear en BlockedAReady")
	slog.Debug("Se lockear en BlockedAReady")
	globals.ESTADOS.BLOCKED = append(globals.ESTADOS.BLOCKED[:pos], globals.ESTADOS.BLOCKED[pos+1:]...)
	globals.ESTADOS.READY = append(globals.ESTADOS.READY, proceso.Pcb.Pid)
	slog.Debug("Se quiere deslockear en BlockedAReady")
	globals.EstadosMutex.Unlock()
	slog.Debug("Se deslockear en BlockedAReady")

	switch globals.KernelConfig.Scheduler_algorithm {
	case "FIFO", "SJF":
		general.Signal(globals.Sem_ProcesosEnReady)
	case "SRT":
		general.NotificarReplanifSRT()
	}

	// LOG Cambio de Estado: ## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado BLOCKED al estado READY", proceso.Pcb.Pid))
}

func BlockedASuspBlocked(proceso globals.Proceso) {
	slog.Debug("Se llego a BlockedASuspBlocked")

	// Muevo el proceso en la colas
	proceso = general.ActualizarMetricas(proceso, proceso.Estado_Actual)
	proceso.Estado_Actual = globals.SUSP_BLOCKED
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso

	slog.Debug("Se actualizo el mapa del proceso")

	slog.Debug("Se quiere lockear en BlockedASuspBlocked")
	globals.EstadosMutex.Lock()
	slog.Debug("Se paso el lock")
	pos := buscarProcesoEnBlocked(proceso.Pcb.Pid)
	globals.ESTADOS.BLOCKED = append(globals.ESTADOS.BLOCKED[:pos], globals.ESTADOS.BLOCKED[pos+1:]...)
	globals.ESTADOS.SUSP_BLOCKED = append(globals.ESTADOS.SUSP_BLOCKED, proceso.Pcb.Pid)
	slog.Debug("Se quiere deslockear en BlockedASuspBlocked")
	globals.EstadosMutex.Unlock()
	slog.Debug("Se desslock")

	// LOG Cambio de Estado: ## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado BLOCKED al estado SUSP_BLOCKED", proceso.Pcb.Pid))

}

func ExecuteABlocked(proceso globals.Proceso, razon string) {
	ahora := time.Now()
	tiempoEnEstado := ahora.Sub(proceso.UltimoCambioDeEstado)
	proceso = general.ActualizarMetricas(proceso, proceso.Estado_Actual)
	if globals.KernelConfig.Scheduler_algorithm != "FIFO" {
		ActualizarEstimado(proceso.Pcb.Pid, float64(tiempoEnEstado.Milliseconds()))
	}

	proceso.Estado_Actual = globals.BLOCKED
	globals.MapaProcesosMutex.Lock()
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso
	globals.MapaProcesosMutex.Unlock()

	slog.Debug("Se quiere lockear en ExecuteABlocked")
	globals.EstadosMutex.Lock()
	slog.Debug("Se lockear en ExecuteABlocked")
	pos := buscarProcesoEnExecute(proceso.Pcb.Pid)
	slog.Debug(fmt.Sprint("Cola EXECUTE: ", globals.ESTADOS.EXECUTE))
	slog.Debug(fmt.Sprint("Cola BLOCKED: ", globals.ESTADOS.BLOCKED))
	globals.ESTADOS.EXECUTE = append(globals.ESTADOS.EXECUTE[:pos], globals.ESTADOS.EXECUTE[pos+1:]...)
	globals.ESTADOS.BLOCKED = append(globals.ESTADOS.BLOCKED, proceso.Pcb.Pid)
	slog.Debug("Se quiere deslockear en ExecuteABlocked")
	globals.EstadosMutex.Unlock()
	slog.Debug("Se deslockear en ExecuteABlocked")

	// LOG Cambio de Estado: ## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado EXECUTE al estado BLOCKED", proceso.Pcb.Pid))
}

func ProcesoAExit(proceso globals.Proceso) {
	// Actualizamos metricas
	proceso = general.ActualizarMetricas(proceso, proceso.Estado_Actual)

	// LOG Cambio de Estado: ## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado %s al estado EXIT", proceso.Pcb.Pid, proceso.Estado_Actual))

	// LOG Fin de Proceso: ## (<PID>) - Finaliza el proceso
	slog.Info(fmt.Sprintf("## (%d) - Finaliza el proceso", proceso.Pcb.Pid))

	// Counts
	newCount := proceso.Pcb.ME.New
	readyCount := proceso.Pcb.ME.Ready
	execCount := proceso.Pcb.ME.Execute
	blockedCount := proceso.Pcb.ME.Blocked
	suspblockedCount := proceso.Pcb.ME.Susp_Blocked
	suspreadyCount := proceso.Pcb.ME.Susp_Ready

	// Times
	newTimes := proceso.Pcb.MT.New.Milliseconds()
	readyTimes := proceso.Pcb.MT.Ready.Milliseconds()
	execTimes := proceso.Pcb.MT.Execute.Milliseconds()
	blockedTimes := proceso.Pcb.MT.Blocked.Milliseconds()
	suspblockedTimes := proceso.Pcb.MT.Susp_Blocked.Milliseconds()
	suspreadyTimes := proceso.Pcb.MT.Susp_Ready.Milliseconds()

	// LOG Métricas de Estado: ## (<PID>) - Métricas de estado: NEW (NEW_COUNT) (NEW_TIME), READY (READY_COUNT) (READY_TIME), …
	slog.Info(fmt.Sprintf("## (%d) - Métricas de estado: NEW %d %dms, READY %d %dms, EXECUTE %d %dms, BLOCKED %d %dms, SUSP_BLOCKED %d %dms, SUSP_READY %d %dms", proceso.Pcb.Pid, newCount, newTimes, readyCount, readyTimes, execCount, execTimes, blockedCount, blockedTimes, suspblockedCount, suspblockedTimes, suspreadyCount, suspreadyTimes))

}

func NewAReady(proceso globals.Proceso_Nuevo) {

	// LOG Cambio de Estado: ## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado NEW al estado READY", proceso.Proceso.Pcb.Pid))

	procesoEnReady := globals.Proceso{
		Pcb:                  proceso.Proceso.Pcb,
		Estado_Actual:        globals.READY,
		Rafaga:               proceso.Proceso.Rafaga,
		UltimoCambioDeEstado: proceso.Proceso.UltimoCambioDeEstado,
	}

	procesoEnReady = general.ActualizarMetricas(procesoEnReady, globals.NEW)
	globals.MapaProcesosMutex.Lock()
	globals.MapaProcesos[procesoEnReady.Pcb.Pid] = procesoEnReady
	globals.MapaProcesosMutex.Unlock()
	slog.Debug("Se quiere lockear en NewAReady")
	globals.EstadosMutex.Lock()
	slog.Debug("Se lockear en NewAReady")
	globals.ESTADOS.READY = append(globals.ESTADOS.READY, procesoEnReady.Pcb.Pid)
	slog.Debug("Se quiere deslockear en NewAReady")
	globals.EstadosMutex.Unlock()
	slog.Debug("Se deslockear en NewAReady")

	//log.Printf("cantidad de procesos en READY: %+v", len(globals.ESTADOS.READY))

	switch globals.KernelConfig.Scheduler_algorithm {
	case "FIFO", "SJF":
		general.Signal(globals.Sem_ProcesosEnReady)
	case "SRT":
		general.NotificarReplanifSRT()
	}
}

func SuspReadyAReady(proceso globals.Proceso) {

	//log.Print("Se quiere loquear MapaProcesos en suspReadyAReady")
	proceso = general.ActualizarMetricas(proceso, proceso.Estado_Actual)
	proceso.Estado_Actual = globals.READY
	globals.MapaProcesosMutex.Lock()
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso
	globals.MapaProcesosMutex.Unlock()
	slog.Debug("Se quiere lockear en SuspReadyAReady")
	globals.EstadosMutex.Lock()
	slog.Debug("Se lockear en SuspReadyAReady")
	globals.ESTADOS.SUSP_READY = globals.ESTADOS.SUSP_READY[1:]
	globals.ESTADOS.READY = append(globals.ESTADOS.READY, proceso.Pcb.Pid)
	slog.Debug("Se quiere deslockear en SuspReadyAReady")
	globals.EstadosMutex.Unlock()
	slog.Debug("Se deslockear en SuspReadyAReady")
	//log.Print("Se unloquea MapaProcesos en suspReadyAReady")

	// LOG Cambio de Estado: ## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado SUSP_READY al estado READY", proceso.Pcb.Pid))

	switch globals.KernelConfig.Scheduler_algorithm {
	case "FIFO", "SJF":
		general.Signal(globals.Sem_ProcesosEnReady)
	case "SRT":
		general.NotificarReplanifSRT()
	}
}

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

func buscarProcesoEnNew(pid int64) int64 {
	colaNew := globals.ESTADOS.NEW

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
	colaSuspBlocked := globals.ESTADOS.SUSP_BLOCKED

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
	colaSuspReady := globals.ESTADOS.SUSP_READY
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
	colaReady := globals.ESTADOS.READY
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
