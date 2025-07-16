package planificadores

import (
	"fmt"
	"log/slog"
	"time"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
	general "github.com/sisoputnfrba/tp-golang/kernel/utils/general"
)

// Se tiene que llamar con Estados Mutex y Mapa Procesos lockeado
func CambiarEstado(pid int64, estadoViejo string, estadoNuevo string) bool {

	var proceso globals.Proceso
	var presente bool
	if estadoViejo != globals.NEW {
		proceso, presente = globals.MapaProcesos[pid]
		if !presente {
			slog.Debug(fmt.Sprintf("PID %d no se encontro en MapaProcesos en CambiarEstado. Posiblemente finalizo", pid))
			return false
		}
	}
	// El resto de transiciones
	var colaVieja *[]int64
	var colaNueva *[]int64

	// Transicion NEW -> READY
	if estadoViejo == globals.NEW && estadoNuevo == globals.READY {
		procesoEnNew := globals.ESTADOS.NEW[0]
		proceso = globals.Proceso{
			Pcb:                  procesoEnNew.Proceso.Pcb,
			Estado_Actual:        globals.READY,
			Rafaga:               procesoEnNew.Proceso.Rafaga,
			UltimoCambioDeEstado: procesoEnNew.Proceso.UltimoCambioDeEstado,
		}

		globals.ESTADOS.NEW = globals.ESTADOS.NEW[1:]
		okey := agregarProcesoACola(&globals.ESTADOS.READY, pid)
		if !okey {
			slog.Debug(fmt.Sprintf("PID %d ya estaba en READY, se evitó duplicación", pid))
		}

		//slog.Debug(fmt.Sprint("COLAS DESPUES DE UN CAMBIO DE ESTADO: "))
		//slog.Debug(fmt.Sprint(estadoViejo, ": ", globals.ESTADOS.NEW))
		//slog.Debug(fmt.Sprint(estadoNuevo, ": ", globals.ESTADOS.READY))
	} else if estadoNuevo == globals.EXIT {

		eliminarProcesoDeSuCola(pid, proceso.Estado_Actual)
		logExit(proceso)

	} else {

		colaVieja = obtenerColaPorEstado(estadoViejo)
		colaNueva = obtenerColaPorEstado(estadoNuevo)

		if !general.EstaEnCola(*colaVieja, pid) {
			slog.Debug(fmt.Sprintf("No se movio PID %d de %s a %s porque ya no estaba en %s", pid, estadoViejo, estadoNuevo, estadoViejo))
			return false
		}

		if general.EstaEnCola(*colaNueva, pid) {
			slog.Debug(fmt.Sprintf("No se movio PID %d de %s a %s porque ya estaba en %s", pid, estadoViejo, estadoNuevo, estadoNuevo))
			return false
		}

		moverEntreColas(proceso, colaVieja, colaNueva)

		//slog.Debug(fmt.Sprint("COLAS DESPUES DE UN CAMBIO DE ESTADO: "))
		//slog.Debug(fmt.Sprint(estadoViejo, ": ", colaVieja))
		//slog.Debug(fmt.Sprint(estadoNuevo, ": ", colaNueva))
	}

	if necesitaReplanificarLargo(estadoViejo, estadoNuevo) {
		globals.DeDondeSeLlamaMutex.Lock()
		globals.DeDondeSeLlamaPasarProcesosAReady = estadoNuevo
		globals.DeDondeSeLlamaMutex.Unlock()
		globals.SignalPasarProcesoAReady()
	}

	if necesitaActualizarEstimado(estadoViejo, estadoNuevo) {
		rafagaReal := float64(time.Since(proceso.UltimoCambioDeEstado).Milliseconds())
		ActualizarEstimado(proceso.Pcb.Pid, float64(rafagaReal))
	}

	proceso = general.ActualizarMetricas(proceso, estadoViejo)
	proceso.Estado_Actual = estadoNuevo
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso

	// LOG Cambio de Estado: ## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado %s al estado %s", pid, estadoViejo, estadoNuevo))

	return true
}

func necesitaActualizarEstimado(estadoViejo string, estadoNuevo string) bool {

	return globals.KernelConfig.Scheduler_algorithm != "FIFO" && (estadoViejo == globals.EXECUTE && estadoNuevo == globals.BLOCKED)
}

func obtenerColaPorEstado(estado string) *[]int64 {
	switch estado {
	case globals.READY:
		return &globals.ESTADOS.READY
	case globals.EXECUTE:
		return &globals.ESTADOS.EXECUTE
	case globals.BLOCKED:
		return &globals.ESTADOS.BLOCKED
	case globals.SUSP_BLOCKED:
		return &globals.ESTADOS.SUSP_BLOCKED
	case globals.SUSP_READY:
		return &globals.ESTADOS.SUSP_READY
	}
	return nil
}

func necesitaReplanificarLargo(estadoViejo string, estadoNuevo string) bool {
	return (estadoViejo == globals.SUSP_BLOCKED && estadoNuevo == globals.SUSP_READY)
}

func moverEntreColas(proceso globals.Proceso, colaVieja *[]int64, colaNueva *[]int64) {

	ok := eliminarProcesoDeCola(colaVieja, proceso.Pcb.Pid)
	if !ok {
		slog.Debug(fmt.Sprintf("PID %d no se encontro en BLOCKED en BlockedASuspBlocked", proceso.Pcb.Pid))
		return
	}

	ok = agregarProcesoACola(colaNueva, proceso.Pcb.Pid)
	if !ok {
		slog.Debug(fmt.Sprintf("PID %d ya estaba en SUSP_BLOCKED, se evitó duplicación", proceso.Pcb.Pid))
	}

}

func agregarProcesoACola(cola *[]int64, pid int64) bool {
	if _, found := BuscarPIDEnCola(*cola, pid); !found {
		*cola = append(*cola, pid)
		return true
	} else {
		slog.Debug(fmt.Sprintf("PID %d ya estaba en la cola, no se agrega de nuevo", pid))
		return false
	}
}

func eliminarProcesoDeCola(cola *[]int64, pid int64) bool {
	pos, found := BuscarPIDEnCola(*cola, pid)
	if found {
		*cola = append((*cola)[:pos], (*cola)[pos+1:]...)
		return true
	} else {
		slog.Debug(fmt.Sprintf("PID %d no se encontro en la cola, no se elimino denuevo", pid))
		return false
	}
}

func BuscarPIDEnCola(cola []int64, pid int64) (int64, bool) {
	for i, valor := range cola {
		if valor == pid {
			return int64(i), true
		}
	}
	return -1, false
}

func BuscarPIDEnNEW(pid int64) (int64, bool) {
	for i, valor := range globals.ESTADOS.NEW {
		if valor.Proceso.Pcb.Pid == pid {
			return int64(i), true
		}
	}
	return -1, false
}

// Busco la cola correspondiente y elimino el proceso
func eliminarProcesoDeSuCola(pid int64, estadoViejo string) bool {
	switch estadoViejo {
	case globals.BLOCKED:
		if eliminarProcesoDeCola(&globals.ESTADOS.BLOCKED, pid) {
			return true
		}
	case globals.EXECUTE:
		if eliminarProcesoDeCola(&globals.ESTADOS.EXECUTE, pid) {
			return true
		}
	case globals.NEW:
		pos, found := BuscarPIDEnNEW(pid)
		if found {
			globals.ESTADOS.NEW = append(globals.ESTADOS.NEW[:pos], globals.ESTADOS.NEW[pos+1:]...)
			return true
		} else {
			slog.Debug(fmt.Sprintf("PID %d no se encontro en NEW en EliminarProcesoDeSuCola", pid))
			return false
		}
	case globals.SUSP_BLOCKED:
		if eliminarProcesoDeCola(&globals.ESTADOS.SUSP_BLOCKED, pid) {
			return true
		}
	case globals.SUSP_READY:
		if eliminarProcesoDeCola(&globals.ESTADOS.SUSP_READY, pid) {
			return true
		}
	case globals.READY:
		if eliminarProcesoDeCola(&globals.ESTADOS.READY, pid) {
			return true
		}
	default:
		slog.Debug(fmt.Sprintf("Error eliminando proceso PID: %d de su cola en EliminarDeSuCola", pid))
		return false
	}
	return false
}

func logExit(proceso globals.Proceso) {

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
