package planificadores

import (
	"fmt"
	"log/slog"
	"time"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
)

// Se tiene que llamar con Estados Mutex y Mapa Procesos lockeado
/*func CambiarEstado(pid int64, estadoViejo string, estadoNuevo string) bool {

	var proceso globals.Proceso
	var presente bool
	var ultimo_cambio_estado time.Time
	if estadoViejo != globals.NEW {
		proceso, presente = globals.MapaProcesos[pid]
		if !presente {
			slog.Debug(fmt.Sprintf("PID %d no se encontro en MapaProcesos en CambiarEstado. Posiblemente finalizo", pid))
			return false
		}
		ultimo_cambio_estado = proceso.UltimoCambioDeEstado
	}

	// El resto de transiciones
	var colaVieja *[]int64
	var colaNueva *[]int64

	// Transicion NEW -> READY
	if estadoViejo == globals.NEW && estadoNuevo == globals.READY {
		procesoEnNew := globals.Cola_new[0]
		proceso = globals.Proceso{
			Pcb:                  procesoEnNew.Proceso.Pcb,
			Estado_Actual:        globals.READY,
			Rafaga:               procesoEnNew.Proceso.Rafaga,
			UltimoCambioDeEstado: procesoEnNew.Proceso.UltimoCambioDeEstado,
		}

		globals.Cola_new = globals.Cola_new[1:]
		okey := agregarProcesoACola(&globals.Cola_ready, pid)
		if !okey {
			slog.Debug(fmt.Sprintf("PID %d ya estaba en READY, se evitó duplicación", pid))
			return false
		}

		proceso = general.ActualizarMetricas(proceso, estadoViejo)
		//slog.Debug(fmt.Sprint("COLAS DESPUES DE UN CAMBIO DE ESTADO: "))
		//slog.Debug(fmt.Sprint(estadoViejo, ": ", globals.ESTADOS.NEW))
		//slog.Debug(fmt.Sprint(estadoNuevo, ": ", globals.ESTADOS.READY))
	} else if estadoNuevo == globals.EXIT {

		slog.Debug(fmt.Sprint("Se llego a CambiarEstado"))
		proceso = general.ActualizarMetricas(proceso, estadoViejo)
		slog.Debug(fmt.Sprint("Actualizadas metricas."))
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

		proceso = general.ActualizarMetricas(proceso, estadoViejo)
		moverEntreColas(proceso, colaVieja, colaNueva)

		//slog.Debug(fmt.Sprint("COLAS DESPUES DE UN CAMBIO DE ESTADO: "))
		//slog.Debug(fmt.Sprint(estadoViejo, ": ", colaVieja))
		//slog.Debug(fmt.Sprint(estadoNuevo, ": ", colaNueva))
	}

	if necesitaActualizarEstimado(estadoViejo, estadoNuevo) {
		rafagaReal := float64(time.Since(ultimo_cambio_estado).Milliseconds())
		ActualizarEstimado(proceso.Pcb.Pid, float64(rafagaReal))
	}
	proceso.Estado_Actual = estadoNuevo
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso

	// LOG Cambio de Estado: ## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado %s al estado %s", pid, estadoViejo, estadoNuevo))

	if necesitaReplanificarLargo(estadoViejo, estadoNuevo) {
		globals.SignalPasarProcesoAReady()
	}

	return true
}*/

func necesitaActualizarEstimado(estadoViejo string, estadoNuevo string) bool {

	return globals.KernelConfig.Scheduler_algorithm != "FIFO" && (estadoViejo == globals.EXECUTE && estadoNuevo == globals.BLOCKED)
}

func obtenerColaPorEstado(estado string) *[]int64 {
	switch estado {
	case globals.READY:
		return &globals.Cola_ready
	case globals.EXECUTE:
		return &globals.Cola_execute
	case globals.BLOCKED:
		return &globals.Cola_blocked
	case globals.SUSP_BLOCKED:
		return &globals.Cola_susp_blocked
	case globals.SUSP_READY:
		return &globals.Cola_susp_ready
	}
	return nil
}

func necesitaReplanificarLargo(estadoViejo string, estadoNuevo string) bool {
	return (estadoViejo == globals.SUSP_BLOCKED && estadoNuevo == globals.SUSP_READY)
}

/*
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

}*/

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

// Busco la cola correspondiente y elimino el proceso
func eliminar_proceso_de(pid int64, estadoViejo string) bool {
	switch estadoViejo {
	case globals.BLOCKED:

		globals.BlockedMutex.Lock()
		if eliminarProcesoDeCola(&globals.Cola_blocked, pid) {
			globals.BlockedMutex.Unlock()
			return true
		}
		globals.BlockedMutex.Unlock()

	case globals.EXECUTE:

		globals.ExecuteMutex.Lock()
		if eliminarProcesoDeCola(&globals.Cola_execute, pid) {
			globals.ExecuteMutex.Unlock()
			return true
		}
		globals.ExecuteMutex.Unlock()

	case globals.NEW:

		globals.NewMutex.Lock()
		if eliminarProcesoDeCola(&globals.Cola_new, pid) {
			globals.NewMutex.Unlock()
			return true
		}
		globals.NewMutex.Unlock()

	case globals.SUSP_BLOCKED:

		globals.SuspBlockedMutex.Lock()
		if eliminarProcesoDeCola(&globals.Cola_susp_blocked, pid) {
			globals.SuspBlockedMutex.Unlock()
			return true
		}
		globals.SuspBlockedMutex.Unlock()

	case globals.SUSP_READY:

		globals.SuspReadyMutex.Lock()
		if eliminarProcesoDeCola(&globals.Cola_susp_ready, pid) {
			globals.SuspReadyMutex.Unlock()
			return true
		}
		globals.SuspReadyMutex.Unlock()

	case globals.READY:

		globals.ReadyMutex.Lock()
		if eliminarProcesoDeCola(&globals.Cola_ready, pid) {
			globals.ReadyMutex.Unlock()
			return true
		}
		globals.ReadyMutex.Unlock()

	default:
		slog.Debug(fmt.Sprintf("Error eliminando proceso PID: %d de su cola en EliminarDeSuCola", pid))
		return false
	}
	return false
}

func ready_a_execute(pid int64) bool {
	globals.ProcesosMutex[pid].Lock()

	globals.ReadyMutex.Lock()
	found := eliminarProcesoDeCola(&globals.Cola_ready, pid)
	if !found {
		return false
	}
	globals.ReadyMutex.Unlock()

	globals.ExecuteMutex.Lock()
	ok := agregarProcesoACola(&globals.Cola_execute, pid)
	if !ok {
		globals.ExecuteMutex.Unlock()
		return false
	}
	globals.ExecuteMutex.Unlock()

	cambiar_estado(pid, globals.READY, globals.EXECUTE)

	globals.ProcesosMutex[pid].Unlock()

	return true
}

func susp_ready_a_ready(pid int64) bool {
	globals.ProcesosMutex[pid].Lock()

	found := eliminarProcesoDeCola(&globals.Cola_susp_ready, pid)
	if !found {
		return false
	}

	globals.ReadyMutex.Lock()
	ok := agregarProcesoACola(&globals.Cola_ready, pid)
	if !ok {
		globals.ReadyMutex.Unlock()
		return false
	}
	globals.ReadyMutex.Unlock()

	cambiar_estado(pid, globals.SUSP_READY, globals.READY)

	globals.ProcesosMutex[pid].Unlock()

	return true
}

func new_a_ready(pid int64) bool {
	globals.ProcesosMutex[pid].Lock()

	found := eliminarProcesoDeCola(&globals.Cola_new, pid)
	if !found {
		return false
	}

	globals.ReadyMutex.Lock()
	ok := agregarProcesoACola(&globals.Cola_ready, pid)
	if !ok {
		globals.ReadyMutex.Unlock()
		return false
	}
	globals.ReadyMutex.Unlock()

	cambiar_estado(pid, globals.NEW, globals.READY)

	globals.ProcesosMutex[pid].Unlock()

	return true
}

func execute_a_blocked(pid int64) bool {
	globals.ProcesosMutex[pid].Lock()

	globals.ExecuteMutex.Lock()
	found := eliminarProcesoDeCola(&globals.Cola_execute, pid)
	if !found {
		globals.ExecuteMutex.Unlock()
		return false
	}
	globals.ExecuteMutex.Unlock()

	globals.BlockedMutex.Lock()
	ok := agregarProcesoACola(&globals.Cola_blocked, pid)
	if !ok {
		globals.BlockedMutex.Unlock()
		return false
	}
	globals.BlockedMutex.Unlock()

	if globals.KernelConfig.Scheduler_algorithm != "FIFO" {

		proceso := globals.MapaProcesos[pid]
		ultimo_cambio_estado := proceso.UltimoCambioDeEstado
		rafagaReal := float64(time.Since(ultimo_cambio_estado).Milliseconds())
		actualizar_rafagas(proceso, rafagaReal)

	}

	cambiar_estado(pid, globals.EXECUTE, globals.BLOCKED)

	globals.ProcesosMutex[pid].Unlock()

	return true
}

func blocked_a_susp_blocked(pid int64) bool {

	globals.BlockedMutex.Lock()
	found := eliminarProcesoDeCola(&globals.Cola_blocked, pid)
	if !found {
		globals.BlockedMutex.Unlock()
		return false
	}
	globals.BlockedMutex.Unlock()

	globals.SuspBlockedMutex.Lock()
	ok := agregarProcesoACola(&globals.Cola_susp_blocked, pid)
	if !ok {
		globals.SuspBlockedMutex.Unlock()
		return false
	}
	globals.SuspBlockedMutex.Unlock()

	cambiar_estado(pid, globals.BLOCKED, globals.SUSP_BLOCKED)

	return true
}

func execute_a_ready(pid int64) bool {

	globals.ExecuteMutex.Lock()
	found := eliminarProcesoDeCola(&globals.Cola_execute, pid)
	if !found {
		globals.ExecuteMutex.Unlock()
		return false
	}
	globals.ExecuteMutex.Unlock()

	globals.ReadyMutex.Lock()
	ok := agregarProcesoACola(&globals.Cola_ready, pid)
	if !ok {
		globals.ReadyMutex.Unlock()
		return false
	}
	globals.ReadyMutex.Unlock()

	cambiar_estado(pid, globals.EXECUTE, globals.READY)

	return true
}

func Blocked_a_ready(pid int64) bool {
	globals.BlockedMutex.Lock()
	found := eliminarProcesoDeCola(&globals.Cola_blocked, pid)
	if !found {
		globals.BlockedMutex.Unlock()
		return false
	}
	globals.BlockedMutex.Unlock()

	globals.ReadyMutex.Lock()
	ok := agregarProcesoACola(&globals.Cola_ready, pid)
	if !ok {
		globals.ReadyMutex.Unlock()
		return false
	}
	globals.ReadyMutex.Unlock()

	cambiar_estado(pid, globals.BLOCKED, globals.READY)

	return true
}

func Susp_blocked_a_Susp_ready(pid int64) bool {
	globals.SuspBlockedMutex.Lock()
	found := eliminarProcesoDeCola(&globals.Cola_susp_blocked, pid)
	if !found {
		globals.SuspBlockedMutex.Unlock()
		return false
	}
	globals.SuspBlockedMutex.Unlock()

	globals.SuspReadyMutex.Lock()
	ok := agregarProcesoACola(&globals.Cola_susp_ready, pid)
	if !ok {
		globals.SuspReadyMutex.Unlock()
		return false
	}
	globals.SuspReadyMutex.Unlock()

	cambiar_estado(pid, globals.SUSP_BLOCKED, globals.SUSP_READY)

	return true
}

func agregar_a_new(pid int64) {

	globals.NewMutex.Lock()
	globals.Cola_new = append(globals.Cola_new, pid)

	//log.Printf("Después de agregar, NEW tiene %d procesos", len(globals.ESTADOS.NEW))
	if globals.KernelConfig.New_algorithm == "PMCP" {
		ordenar_new()
	}
	globals.NewMutex.Unlock()

	// LOG Creación de Proceso: "## (<PID>) Se crea el proceso - Estado: NEW"
	slog.Info(fmt.Sprintf("## (%d) Se crea el proceso - Estado: NEW", pid))

}

func actualizar_metricas(pid int64, estadoAnterior string) {

	proceso, ok := globals.MapaProcesos[pid]
	if !ok {
		return
	}

	ahora := time.Now()
	tiempoEnEstado := ahora.Sub(proceso.UltimoCambioDeEstado)

	switch estadoAnterior {
	case globals.NEW:
		proceso.Pcb.ME.New++
		proceso.Pcb.MT.New += tiempoEnEstado
	case globals.READY:
		proceso.Pcb.ME.Ready++
		proceso.Pcb.MT.Ready += tiempoEnEstado
	case globals.EXECUTE:
		proceso.Pcb.ME.Execute++
		proceso.Pcb.MT.Execute += tiempoEnEstado
	case globals.BLOCKED:
		proceso.Pcb.ME.Blocked++
		proceso.Pcb.MT.Blocked += tiempoEnEstado
	case globals.SUSP_BLOCKED:
		proceso.Pcb.ME.Susp_Blocked++
		proceso.Pcb.MT.Susp_Blocked += tiempoEnEstado
	case globals.SUSP_READY:
		proceso.Pcb.ME.Susp_Ready++
		proceso.Pcb.MT.Susp_Ready += tiempoEnEstado
	default:
	}
	proceso.UltimoCambioDeEstado = ahora

}

func cambiar_estado(pid int64, estado_viejo string, estado_nuevo string) {
	proceso := globals.MapaProcesos[pid]
	proceso.Estado_Actual = estado_nuevo

	actualizar_metricas(pid, estado_viejo)

	// LOG Cambio de Estado: ## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado %s al estado %s", pid, estado_viejo, estado_nuevo))
}
