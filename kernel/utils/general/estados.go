package utils_general

/*
func SuspBlockedASuspReady(proceso globals.Proceso) {
	//log.Print("Se quiere loquear MapaProcesos en SuspBlockedASuspReady")
	globals.MapaProcesosMutex.Lock()
	proceso = ActualizarMetricas(proceso, proceso.Estado_Actual)
	proceso.Estado_Actual = globals.SUSP_READY
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso
	globals.MapaProcesosMutex.Unlock()
	//log.Print("Se unloquea MapaProcesos en SuspBlockedASuspReady")

	pos := BuscarProcesoEnSuspBlocked(proceso.Pcb.Pid)

	globals.EstadosMutex.Lock()
	globals.ESTADOS.SUSP_BLOCKED = append(globals.ESTADOS.SUSP_BLOCKED[:pos], globals.ESTADOS.SUSP_BLOCKED[pos+1:]...)
	globals.ESTADOS.SUSP_READY = append(globals.ESTADOS.SUSP_READY, proceso.Pcb.Pid)
	globals.EstadosMutex.Unlock()

	log.Printf("Proceso de PID %d fue movido de Susp Blocked a Susp Ready", proceso.Pcb.Pid)

	globals.DeDondeSeLlamaMutex.Lock()
	globals.DeDondeSeLlamaPasarProcesosAReady = "SUSP READY"
	globals.DeDondeSeLlamaMutex.Unlock()
	Signal(globals.Sem_PasarProcesoAReady)

	// LOG Cambio de Estado: ## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado SUSP_BLOCKED al estado SUSP_READY", proceso.Pcb.Pid))
}

func BlockedAReady(proceso globals.Proceso) {
	//log.Print("Se quiere loquear MapaProcesos en BlockedAReady")
	globals.MapaProcesosMutex.Lock()
	proceso = ActualizarMetricas(proceso, proceso.Estado_Actual)
	proceso.Estado_Actual = globals.READY
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso
	globals.MapaProcesosMutex.Unlock()
	//log.Print("Se unloquea MapaProcesos en BlockedAReady")

	pos := BuscarProcesoEnBlocked(proceso.Pcb.Pid)

	globals.EstadosMutex.Lock()
	globals.ESTADOS.BLOCKED = append(globals.ESTADOS.BLOCKED[:pos], globals.ESTADOS.BLOCKED[pos+1:]...)
	globals.ESTADOS.READY = append(globals.ESTADOS.SUSP_READY, proceso.Pcb.Pid)
	globals.EstadosMutex.Unlock()

	log.Printf("Proceso de PID %d fue movido de Blocked a Ready", proceso.Pcb.Pid)

	log.Printf("cantidad de procesos en READY: %+v", len(globals.ESTADOS.READY))

	switch globals.KernelConfig.Scheduler_algorithm {
	case "FIFO", "SJF":
		Signal(globals.Sem_ProcesosEnReady)
	case "SRT":
		NotificarReplanifSRT()
	}

	// LOG Cambio de Estado: ## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado BLOCKED al estado READY", proceso.Pcb.Pid))
}

func BlockedASuspBlocked(proceso globals.Proceso) {
	// Muevo el proceso en la colas
	//log.Print("Se quiere loquear MapaProcesos en blockedASuspBlocked")
	globals.MapaProcesosMutex.Lock()
	proceso = ActualizarMetricas(proceso, proceso.Estado_Actual)
	proceso.Estado_Actual = globals.SUSP_BLOCKED
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso
	globals.MapaProcesosMutex.Unlock()
	//log.Print("Se unloquea MapaProcesos en blockedASuspBlocked")

	globals.EstadosMutex.Lock()
	pos := BuscarProcesoEnBlocked(proceso.Pcb.Pid)
	globals.ESTADOS.BLOCKED = append(globals.ESTADOS.BLOCKED[:pos], globals.ESTADOS.BLOCKED[pos+1:]...)
	globals.ESTADOS.SUSP_BLOCKED = append(globals.ESTADOS.SUSP_BLOCKED, proceso.Pcb.Pid)
	globals.EstadosMutex.Unlock()

	// LOG Cambio de Estado: ## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado BLOCKED al estado SUSP_BLOCKED", proceso.Pcb.Pid))

}

func ExecuteABlocked(proceso globals.Proceso, razon string) {
	globals.MapaProcesosMutex.Lock()
	proceso = ActualizarMetricas(proceso, proceso.Estado_Actual)
	proceso.Estado_Actual = globals.BLOCKED
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso
	globals.MapaProcesosMutex.Unlock()

	globals.EstadosMutex.Lock()
	pos := BuscarProcesoEnExecute(proceso.Pcb.Pid)
	globals.ESTADOS.EXECUTE = append(globals.ESTADOS.EXECUTE[:pos], globals.ESTADOS.EXECUTE[pos+1:]...)
	globals.ESTADOS.BLOCKED = append(globals.ESTADOS.BLOCKED, proceso.Pcb.Pid)
	globals.EstadosMutex.Unlock()

	// LOG Cambio de Estado: ## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado EXECUTE al estado BLOCKED", proceso.Pcb.Pid))
}*/

// Se llama con estados mutex lockeado
