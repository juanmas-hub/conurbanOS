package estados

import (
	"fmt"
	"log"
	"log/slog"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
	general "github.com/sisoputnfrba/tp-golang/kernel/utils/general"
)

func SuspBlockedASuspReady(proceso globals.Proceso) {
	//log.Print("Se quiere loquear MapaProcesos en SuspBlockedASuspReady")
	proceso = general.ActualizarMetricas(proceso, proceso.Estado_Actual)
	proceso.Estado_Actual = globals.SUSP_READY
	globals.MapaProcesosMutex.Lock()
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso
	globals.MapaProcesosMutex.Unlock()
	//log.Print("Se unloquea MapaProcesos en SuspBlockedASuspReady")

	pos := buscarProcesoEnSuspBlocked(proceso.Pcb.Pid)

	globals.EstadosMutex.Lock()
	globals.ESTADOS.SUSP_BLOCKED = append(globals.ESTADOS.SUSP_BLOCKED[:pos], globals.ESTADOS.SUSP_BLOCKED[pos+1:]...)
	globals.ESTADOS.SUSP_READY = append(globals.ESTADOS.SUSP_READY, proceso.Pcb.Pid)
	globals.EstadosMutex.Unlock()

	log.Printf("Proceso de PID %d fue movido de Susp Blocked a Susp Ready", proceso.Pcb.Pid)

	globals.DeDondeSeLlamaMutex.Lock()
	globals.DeDondeSeLlamaPasarProcesosAReady = "SUSP READY"
	globals.DeDondeSeLlamaMutex.Unlock()
	//general.Signal(globals.Sem_PasarProcesoAReady)
	globals.SignalPasarProcesoAReady()

	// LOG Cambio de Estado: ## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado SUSP_BLOCKED al estado SUSP_READY", proceso.Pcb.Pid))
}

func BlockedAReady(proceso globals.Proceso) {
	//log.Print("Se quiere loquear MapaProcesos en BlockedAReady")
	proceso = general.ActualizarMetricas(proceso, proceso.Estado_Actual)
	proceso.Estado_Actual = globals.READY
	globals.MapaProcesosMutex.Lock()
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso
	globals.MapaProcesosMutex.Unlock()
	//log.Print("Se unloquea MapaProcesos en BlockedAReady")

	pos := buscarProcesoEnBlocked(proceso.Pcb.Pid)

	//log.Print("Se quiere bloquear en BlockedAReady")
	globals.EstadosMutex.Lock()
	//log.Print("Se bloqueo en BlockedAReady")
	globals.ESTADOS.BLOCKED = append(globals.ESTADOS.BLOCKED[:pos], globals.ESTADOS.BLOCKED[pos+1:]...)
	globals.ESTADOS.READY = append(globals.ESTADOS.SUSP_READY, proceso.Pcb.Pid)
	//log.Print("Se quiere desbloquear en BlockedAReady")
	globals.EstadosMutex.Unlock()
	//log.Print("Se desbloqueo en BlockedAReady")

	log.Printf("Proceso de PID %d fue movido de Blocked a Ready", proceso.Pcb.Pid)

	log.Printf("cantidad de procesos en READY: %+v", len(globals.ESTADOS.READY))

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
	// Muevo el proceso en la colas
	//log.Print("Se quiere loquear MapaProcesos en blockedASuspBlocked")
	proceso = general.ActualizarMetricas(proceso, proceso.Estado_Actual)
	proceso.Estado_Actual = globals.SUSP_BLOCKED
	globals.MapaProcesosMutex.Lock()
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso
	globals.MapaProcesosMutex.Unlock()
	//log.Print("Se unloquea MapaProcesos en blockedASuspBlocked")

	globals.EstadosMutex.Lock()
	pos := buscarProcesoEnBlocked(proceso.Pcb.Pid)
	globals.ESTADOS.BLOCKED = append(globals.ESTADOS.BLOCKED[:pos], globals.ESTADOS.BLOCKED[pos+1:]...)
	globals.ESTADOS.SUSP_BLOCKED = append(globals.ESTADOS.SUSP_BLOCKED, proceso.Pcb.Pid)
	globals.EstadosMutex.Unlock()

	// LOG Cambio de Estado: ## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado BLOCKED al estado SUSP_BLOCKED", proceso.Pcb.Pid))

}

func ExecuteABlocked(proceso globals.Proceso, razon string) {
	proceso = general.ActualizarMetricas(proceso, proceso.Estado_Actual)
	proceso.Estado_Actual = globals.BLOCKED
	globals.MapaProcesosMutex.Lock()
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso
	globals.MapaProcesosMutex.Unlock()

	//log.Print("Se quiere bloquear en ExecuteABlocked")
	globals.EstadosMutex.Lock()
	//log.Print("Se bloqueo en ExecuteABlocked")
	pos := buscarProcesoEnExecute(proceso.Pcb.Pid)
	globals.ESTADOS.EXECUTE = append(globals.ESTADOS.EXECUTE[:pos], globals.ESTADOS.EXECUTE[pos+1:]...)
	globals.ESTADOS.BLOCKED = append(globals.ESTADOS.BLOCKED, proceso.Pcb.Pid)
	//log.Print("Se quiere desbloquear en ExecuteABlocked")
	globals.EstadosMutex.Unlock()
	//log.Print("Se desbloqueo en ExecuteABlocked")

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
	globals.EstadosMutex.Lock()
	globals.ESTADOS.NEW = globals.ESTADOS.NEW[1:]
	globals.ESTADOS.READY = append(globals.ESTADOS.READY, procesoEnReady.Pcb.Pid)
	globals.EstadosMutex.Unlock()

	log.Printf("cantidad de procesos en READY: %+v", len(globals.ESTADOS.READY))

	switch globals.KernelConfig.Scheduler_algorithm {
	case "FIFO", "SJF":
		general.Signal(globals.Sem_ProcesosEnReady)
	case "SRT":
		general.NotificarReplanifSRT()
	}

	// LOG Cambio de Estado: ## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado NEW al estado READY", procesoEnReady.Pcb.Pid))
}

func SuspReadyAReady(proceso globals.Proceso) {

	//log.Print("Se quiere loquear MapaProcesos en suspReadyAReady")
	proceso = general.ActualizarMetricas(proceso, proceso.Estado_Actual)
	proceso.Estado_Actual = globals.READY
	globals.MapaProcesosMutex.Lock()
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso
	globals.MapaProcesosMutex.Unlock()
	globals.EstadosMutex.Lock()
	globals.ESTADOS.SUSP_READY = globals.ESTADOS.SUSP_READY[1:]
	globals.ESTADOS.READY = append(globals.ESTADOS.READY, proceso.Pcb.Pid)
	globals.EstadosMutex.Unlock()
	//log.Print("Se unloquea MapaProcesos en suspReadyAReady")

	switch globals.KernelConfig.Scheduler_algorithm {
	case "FIFO", "SJF":
		general.Signal(globals.Sem_ProcesosEnReady)
	case "SRT":
		general.NotificarReplanifSRT()
	}

	// LOG Cambio de Estado: ## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado SUSP_READY al estado READY", proceso.Pcb.Pid))
}
