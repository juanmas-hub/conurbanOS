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

	globals.MapaProcesosMutex.Lock()
	defer globals.MapaProcesosMutex.Unlock()
	general.LogLockeo("Mapa Procesos", "SuspBlockedASuspReady")
	defer general.LogUnlockeo("Mapa Procesos", "SuspBlockedASuspReady")

	globals.EstadosMutex.Lock()
	defer globals.EstadosMutex.Unlock()

	proceso = general.ActualizarMetricas(proceso, proceso.Estado_Actual)
	proceso.Estado_Actual = globals.SUSP_READY
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso

	ok := EliminarProcesoDeCola(&globals.ESTADOS.SUSP_BLOCKED, proceso.Pcb.Pid)
	if !ok {
		slog.Debug(fmt.Sprintf("PID %d no se encontro en SUSP_BLOCKED en SuspBlockedASuspReady", proceso.Pcb.Pid))
		return
	}

	ok = AgregarProcesoACola(&globals.ESTADOS.SUSP_READY, proceso.Pcb.Pid)
	if !ok {
		slog.Debug(fmt.Sprintf("PID %d ya estaba en SUSP_READY, se evitó duplicación", proceso.Pcb.Pid))
	}

	globals.DeDondeSeLlamaMutex.Lock()
	globals.DeDondeSeLlamaPasarProcesosAReady = "SUSP READY"
	globals.DeDondeSeLlamaMutex.Unlock()
	globals.SignalPasarProcesoAReady()

	// LOG Cambio de Estado: ## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado SUSP_BLOCKED al estado SUSP_READY", proceso.Pcb.Pid))
}

func BlockedAReady(proceso globals.Proceso) {

	globals.MapaProcesosMutex.Lock()
	defer globals.MapaProcesosMutex.Unlock()
	general.LogLockeo("Mapa Procesos", "BlockedAReady")
	defer general.LogUnlockeo("Mapa Procesos", "BlockedAReady")

	globals.EstadosMutex.Lock()
	defer globals.EstadosMutex.Unlock()

	slog.Debug(fmt.Sprint("Se quiere pasar de BLOCKED a READY: ", proceso.Pcb.Pid))

	proceso = general.ActualizarMetricas(proceso, proceso.Estado_Actual)
	proceso.Estado_Actual = globals.READY
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso

	ok := EliminarProcesoDeCola(&globals.ESTADOS.BLOCKED, proceso.Pcb.Pid)
	if !ok {
		slog.Debug(fmt.Sprintf("PID %d no se encontro en BLOCKED en BlockedAReady", proceso.Pcb.Pid))
		return
	}

	ok = AgregarProcesoACola(&globals.ESTADOS.READY, proceso.Pcb.Pid)
	if !ok {
		slog.Debug(fmt.Sprintf("PID %d ya estaba en READY, se evitó duplicación", proceso.Pcb.Pid))
	}

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

	globals.MapaProcesosMutex.Lock()
	defer globals.MapaProcesosMutex.Unlock()
	general.LogLockeo("Mapa Procesos", "BlockedASuspBlocked")
	defer general.LogUnlockeo("Mapa Procesos", "BlockedASuspBlocked")

	globals.EstadosMutex.Lock()
	defer globals.EstadosMutex.Unlock()

	// Muevo el proceso en la colas
	proceso = general.ActualizarMetricas(proceso, globals.BLOCKED)
	proceso.Estado_Actual = globals.SUSP_BLOCKED
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso

	slog.Debug(fmt.Sprint("Se quiere pasar de BLOCKED a SUSP_BLOCKED: ", proceso.Pcb.Pid))

	ok := EliminarProcesoDeCola(&globals.ESTADOS.BLOCKED, proceso.Pcb.Pid)
	if !ok {
		slog.Debug(fmt.Sprintf("PID %d no se encontro en BLOCKED en BlockedASuspBlocked", proceso.Pcb.Pid))
		return
	}

	ok = AgregarProcesoACola(&globals.ESTADOS.SUSP_BLOCKED, proceso.Pcb.Pid)
	if !ok {
		slog.Debug(fmt.Sprintf("PID %d ya estaba en SUSP_BLOCKED, se evitó duplicación", proceso.Pcb.Pid))
	}

	// LOG Cambio de Estado: ## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado BLOCKED al estado SUSP_BLOCKED", proceso.Pcb.Pid))

}

func ExecuteABlocked(proceso globals.Proceso, razon string) {

	globals.MapaProcesosMutex.Lock()
	defer globals.MapaProcesosMutex.Unlock()
	general.LogLockeo("Mapa Procesos", "ExecuteABlocked")
	defer general.LogUnlockeo("Mapa Procesos", "ExecuteABlocked")

	if general.EstaEnCola(globals.ESTADOS.BLOCKED, proceso.Pcb.Pid) {
		slog.Debug(fmt.Sprintf("ExecuteABlocked ignorado: PID %d ya estaba en BLOCKED", proceso.Pcb.Pid))
		return
	}

	globals.EstadosMutex.Lock()
	defer globals.EstadosMutex.Unlock()

	ahora := time.Now()
	tiempoEnEstado := ahora.Sub(proceso.UltimoCambioDeEstado)
	proceso = general.ActualizarMetricas(proceso, globals.EXECUTE)
	slog.Debug(fmt.Sprint("Proceso en ExecuteABlocked: ", proceso))
	if globals.KernelConfig.Scheduler_algorithm != "FIFO" {
		ActualizarEstimado(proceso.Pcb.Pid, float64(tiempoEnEstado.Milliseconds()))
	}

	slog.Debug(fmt.Sprint("Se actualizo el estimado en ExecuteABlocked"))

	proceso.Estado_Actual = globals.BLOCKED
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso

	slog.Debug(fmt.Sprint("Se quiere pasar de EXECUTE a BLOCKED: ", proceso.Pcb.Pid))

	ok := EliminarProcesoDeCola(&globals.ESTADOS.EXECUTE, proceso.Pcb.Pid)
	if !ok {
		slog.Debug(fmt.Sprintf("PID %d no se encontro en EXECUTE en ExecuteABlocked", proceso.Pcb.Pid))
		return
	}

	ok = AgregarProcesoACola(&globals.ESTADOS.BLOCKED, proceso.Pcb.Pid)
	if !ok {
		slog.Debug(fmt.Sprintf("PID %d ya estaba en BLOCKED, se evitó duplicación", proceso.Pcb.Pid))
	}

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
	general.LogIntentoLockeo("Mapa Procesos", "NewAReady")
	globals.MapaProcesosMutex.Lock()
	general.LogLockeo("Mapa Procesos", "NewAReady")
	globals.MapaProcesos[procesoEnReady.Pcb.Pid] = procesoEnReady
	globals.MapaProcesosMutex.Unlock()
	general.LogUnlockeo("Mapa Procesos", "NewAReady")

	globals.EstadosMutex.Lock()
	okey := AgregarProcesoACola(&globals.ESTADOS.READY, proceso.Proceso.Pcb.Pid)
	if !okey {
		slog.Debug(fmt.Sprintf("PID %d ya estaba en READY, se evitó duplicación", proceso.Proceso.Pcb.Pid))
	}
	globals.EstadosMutex.Unlock()

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
	general.LogIntentoLockeo("Mapa Procesos", "SuspReadyAReady")
	globals.MapaProcesosMutex.Lock()
	general.LogLockeo("Mapa Procesos", "SuspReadyAReady")
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso
	globals.MapaProcesosMutex.Unlock()
	general.LogUnlockeo("Mapa Procesos", "SuspReadyAReady")

	globals.EstadosMutex.Lock()
	ok := EliminarProcesoDeCola(&globals.ESTADOS.SUSP_READY, proceso.Pcb.Pid)
	if !ok {
		slog.Debug(fmt.Sprintf("PID %d no se encontro en SUSP_READY en SuspReadyAReady", proceso.Pcb.Pid))
		return
	}

	ok = AgregarProcesoACola(&globals.ESTADOS.READY, proceso.Pcb.Pid)
	if !ok {
		slog.Debug(fmt.Sprintf("PID %d ya estaba en READY, se evitó duplicación", proceso.Pcb.Pid))
	}
	globals.EstadosMutex.Unlock()

	// LOG Cambio de Estado: ## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado SUSP_READY al estado READY", proceso.Pcb.Pid))

	switch globals.KernelConfig.Scheduler_algorithm {
	case "FIFO", "SJF":
		general.Signal(globals.Sem_ProcesosEnReady)
	case "SRT":
		general.NotificarReplanifSRT()
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

func AgregarProcesoACola(cola *[]int64, pid int64) bool {
	if _, found := BuscarPIDEnCola(*cola, pid); !found {
		*cola = append(*cola, pid)
		return true
	} else {
		slog.Debug(fmt.Sprintf("PID %d ya estaba en la cola, no se agrega de nuevo", pid))
		return false
	}
}

func EliminarProcesoDeCola(cola *[]int64, pid int64) bool {
	pos, found := BuscarPIDEnCola(*cola, pid)
	if found {
		*cola = append((*cola)[:pos], (*cola)[pos+1:]...)
		return true
	} else {
		slog.Debug(fmt.Sprintf("PID %d no se encontro en la cola, no se elimino denuevo", pid))
		return false
	}
}

// Busco la cola correspondiente y elimino el proceso
func EliminarProcesoDeSuCola(pid int64, estadoActual string) bool {
	switch estadoActual {
	case globals.BLOCKED:
		if EliminarProcesoDeCola(&globals.ESTADOS.BLOCKED, pid) {
			return true
		}
	case globals.EXECUTE:
		if EliminarProcesoDeCola(&globals.ESTADOS.BLOCKED, pid) {
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
		if EliminarProcesoDeCola(&globals.ESTADOS.BLOCKED, pid) {
			return true
		}
	case globals.SUSP_READY:
		if EliminarProcesoDeCola(&globals.ESTADOS.BLOCKED, pid) {
			return true
		}
	case globals.READY:
		if EliminarProcesoDeCola(&globals.ESTADOS.BLOCKED, pid) {
			return true
		}
	default:
		log.Printf("Error eliminando proceso PID: %d de su cola en EliminarDeSuCola", pid)
		return false
	}
	return false
}
