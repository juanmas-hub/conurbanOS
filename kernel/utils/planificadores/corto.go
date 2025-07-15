package planificadores

import (
	"fmt"
	"log"
	"log/slog"
	"sort"
	"time"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
	general "github.com/sisoputnfrba/tp-golang/kernel/utils/general"
)

func EjecutarPlanificadorCortoPlazo() {

	if globals.KernelConfig.Scheduler_algorithm == "FIFO" {

		go planificadorFIFO()

	}

	if globals.KernelConfig.Scheduler_algorithm == "SJF" {

		go planificadorSJF()

	}

	if globals.KernelConfig.Scheduler_algorithm == "SRT" {

		go planificadorSRT()

	}

}

func planificadorFIFO() {
	for {
		general.Wait(globals.Sem_Cpus)            // Espero a que haya Cpus libres
		general.Wait(globals.Sem_ProcesosEnReady) // Espero a que haya procesos en Ready

		//slog.Debug("Se quiere lockear en planificadorFIFO")
		globals.EstadosMutex.Lock()
		//slog.Debug("Se lockear en planificadorFIFO")
		ejecutarUnProceso()

		//slog.Debug("Se quiere deslockear en planificadorFIFO")
		globals.EstadosMutex.Unlock()
		//slog.Debug("Se deslockear en planificadorFIFO")
	}
}

func planificadorSJF() {
	for {
		general.Wait(globals.Sem_Cpus)
		general.Wait(globals.Sem_ProcesosEnReady)

		globals.EstadosMutex.Lock()

		ordenarReadyPorRafaga()
		ejecutarUnProceso()

		globals.EstadosMutex.Unlock()
	}
}

func planificadorSRT() {
	for {
		//slog.Debug(fmt.Sprint("Esperando Replanificar SRT"))
		<-globals.SrtReplanificarChan
		//slog.Debug(fmt.Sprint("Se llamó para Replanificar SRT"))

		// Chequeo los 4 posibles casos

		if hayProcesosEnReady() && hayCpusLibres() {
			globals.EstadosMutex.Lock()

			slog.Debug(fmt.Sprint("SRT - Hay procesos en ready y hay cpus libres"))

			ordenarReadyPorRafaga()
			ejecutarUnProceso()

			globals.EstadosMutex.Unlock()
		}

		if hayProcesosEnReady() && !hayCpusLibres() {

			slog.Debug(fmt.Sprint("SRT - Hay procesos en ready y NO hay cpus libres"))
			// Caso desalojo
			pidEnExec, hayQueDesalojar := verificarDesalojo()
			if hayQueDesalojar {
				slog.Debug(fmt.Sprint("SRT - PID elegido a deslojar: ", pidEnExec))
				desalojarYEnviarProceso(pidEnExec)
			}
		}

		if !hayProcesosEnReady() && hayCpusLibres() {
			slog.Debug(fmt.Sprint("SRT - No hay procesos en ready y hay cpus libres"))
			// No hacemos nada
		}

		if !hayProcesosEnReady() && !hayCpusLibres() {
			slog.Debug(fmt.Sprint("SRT - No hay ni procesos en ready ni cpus libres"))
			// No hacemos nada
		}
	}
}

func ActualizarEstimado(pid int64, rafagaReal float64) {
	// Me imagino que esto se usa cuando se termina de ejecutar un proceso

	slog.Debug(fmt.Sprintf("Rafaga real: %f", rafagaReal))

	proceso := globals.MapaProcesos[pid]

	alpha := globals.KernelConfig.Alpha
	est_ant := proceso.Rafaga.Est_Sgte

	proceso.Rafaga.Est_Ant = est_ant
	proceso.Rafaga.Raf_Ant = rafagaReal
	proceso.Rafaga.Est_Sgte = rafagaReal*alpha + est_ant*(1-alpha)
	// Est(n+1) =  R(n) + (1-) Est(n) ;    [0,1]

	slog.Debug(fmt.Sprintf("Rafaga actualizada de PID %d: %f", proceso.Pcb.Pid, proceso.Rafaga))

	globals.MapaProcesos[pid] = proceso
}

func elegirCPUlibre() (string, int64, string) {
	globals.ListaCPUsMutex.Lock()
	encontrado := false
	var cpu globals.ListaCpu
	// Recorremos la lista
	for i := range globals.ListaCPUs {
		// Si la posicion i esta libre
		if globals.ListaCPUs[i].EstaLibre {
			// La marcamos como ocupada
			globals.ListaCPUs[i].EstaLibre = false
			cpu = globals.ListaCPUs[i]
			encontrado = true
			break
		}
	}

	globals.ListaCPUsMutex.Unlock()
	// Devolvemos IP y PUERTO
	if encontrado {
		return cpu.Handshake.IP, cpu.Handshake.Puerto, cpu.Handshake.Nombre
	} else {
		// Si devuelve esto hay un error, porque esta funcion se tiene que ejecutar cuando el semaforo lo permita
		log.Println("No se encontro CPU libre")
		return "", -1, ""
	}
}

func aExecute(proceso globals.Proceso) {
	// Esto funcionaría para FIFO y SJF. Nose si SRT

	estado_anterior := proceso.Estado_Actual

	proceso = general.ActualizarMetricas(proceso, proceso.Estado_Actual)
	proceso.Estado_Actual = globals.EXECUTE
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso
	ok := EliminarProcesoDeCola(&globals.ESTADOS.READY, proceso.Pcb.Pid)
	if !ok {
		slog.Debug(fmt.Sprintf("PID %d no se encontro en READY en aExecute", proceso.Pcb.Pid))
		return
	}

	ok = AgregarProcesoACola(&globals.ESTADOS.EXECUTE, proceso.Pcb.Pid)
	if !ok {
		slog.Debug(fmt.Sprintf("PID %d ya estaba en EXECUTE, se evitó duplicación", proceso.Pcb.Pid))
		return
	}

	// LOG Cambio de Estado: ## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado %s al estado EXECUTE", proceso.Pcb.Pid, estado_anterior))
}

func hayProcesosEnReady() bool {
	return len(globals.ESTADOS.READY) > 0
}

func hayCpusLibres() bool {
	globals.ListaCPUsMutex.Lock()
	defer globals.ListaCPUsMutex.Unlock()
	for _, cpu := range globals.ListaCPUs {
		if cpu.EstaLibre {
			return true
		}
	}
	return false
}

// Chequea si hay que desalojar. Si hay que desalojar, devuelve el PID que esta ejecutando
func verificarDesalojo() (int64, bool) {

	globals.EstadosMutex.Lock()
	defer globals.EstadosMutex.Unlock()
	globals.MapaProcesosMutex.Lock()
	defer globals.MapaProcesosMutex.Unlock()
	general.LogLockeo("Mapa Procesos", "verificarDesalojo")
	defer general.LogUnlockeo("Mapa Procesos", "verificarDesalojo")

	ordenarReadyPorRafaga()
	pidEnExec, restanteExec, encontro := buscarProcesoEnExecuteDeMenorRafagaRestante()

	if encontro {
		rafagaNuevo := globals.MapaProcesos[globals.ESTADOS.READY[0]].Rafaga.Est_Sgte

		if rafagaNuevo < restanteExec {
			return pidEnExec, true
		}
	}

	return -1, false

}

func buscarProcesoEnExecuteDeMenorRafagaRestante() (int64, float64, bool) {
	slog.Debug(fmt.Sprint("Buscando menor rafaga restante en EXECUTE: ", globals.ESTADOS.EXECUTE))

	var pidMenorRafaga int64
	var menorRafagaRestante float64
	encontro := false

	for _, pidActual := range globals.ESTADOS.EXECUTE {
		_, ok := globals.MapaProcesos[pidActual]
		if !ok {
			slog.Warn(fmt.Sprintf("PID %d no encontrado en MapaProcesos. Posiblemente finalizado.", pidActual))
			continue
		}

		restante := rafagaRestante(pidActual)
		if !encontro || restante < menorRafagaRestante {
			pidMenorRafaga = pidActual
			menorRafagaRestante = restante
			encontro = true
		}
	}

	if !encontro {
		slog.Warn("No se encontró ningún proceso válido en EXECUTE para evaluar desalojo.")
		return 0, 0, false
	}

	slog.Debug(fmt.Sprintf("PID de menor rafaga restante: %d, restante: %f", pidMenorRafaga, menorRafagaRestante))
	return pidMenorRafaga, menorRafagaRestante, true
}

func rafagaRestante(pid int64) float64 {

	slog.Debug(fmt.Sprint("PID: ", pid))
	proceso := globals.MapaProcesos[pid]
	slog.Debug(fmt.Sprint("Proceso: ", proceso))
	rafaga := proceso.Rafaga.Est_Sgte
	slog.Debug(fmt.Sprint("Rafaga: ", rafaga))                                       // float64
	tiempoPasado := float64(time.Since(proceso.UltimoCambioDeEstado).Milliseconds()) // tiempo en ms
	slog.Debug(fmt.Sprint("Tiempo pasado: ", tiempoPasado))

	slog.Debug(fmt.Sprint("Rafaga restante: ", rafaga-tiempoPasado))
	return rafaga - tiempoPasado
}

func ordenarReadyPorRafaga() {
	// sort.SLice compara pares de elementos (i y j) si i < j -> true, si j < i -> false
	sort.Slice(globals.ESTADOS.READY, func(i, j int) bool {
		pidI := globals.ESTADOS.READY[i]
		pidJ := globals.ESTADOS.READY[j]

		// De cada par de procesos sacamos la rafaga que tiene cada uno
		rafagaI := globals.MapaProcesos[pidI].Rafaga
		rafagaJ := globals.MapaProcesos[pidJ].Rafaga
		// Si la rafagaI es menor, la ponemos antes
		return rafagaI.Est_Sgte < rafagaJ.Est_Sgte
	})

	var rafagasReady []float64
	for i := range globals.ESTADOS.READY {
		// Si la posicion i esta libre
		pidActual := globals.ESTADOS.READY[i]
		rafagasReady = append(rafagasReady, globals.MapaProcesos[pidActual].Rafaga.Est_Sgte)
	}
	slog.Debug(fmt.Sprintf("Ready ordenado por rafaga: %d, %f", globals.ESTADOS.READY, rafagasReady))

}

func ejecutarUnProceso() {
	slog.Debug(fmt.Sprintf(" ---- Ready: %d ----", globals.ESTADOS.READY))
	procesoAEjecutar := globals.ESTADOS.READY[0]
	ip, port, nombre := elegirCPUlibre()
	general.LogIntentoLockeo("Mapa Procesos", "ejecutarUnProceso")
	globals.MapaProcesosMutex.Lock()
	general.LogLockeo("Mapa Procesos", "ejecutarUnProceso")
	proceso := globals.MapaProcesos[procesoAEjecutar]
	general.EnviarProcesoAEjecutar_ACPU(ip, port, proceso.Pcb.Pid, proceso.Pcb.PC, nombre)
	aExecute(proceso)
	globals.MapaProcesosMutex.Unlock()
	general.LogUnlockeo("Mapa Procesos", "ejecutarUnProceso")
}

func desalojarYEnviarProceso(pidEnExec int64) {
	globals.EstadosMutex.Lock()
	defer globals.EstadosMutex.Unlock()
	globals.MapaProcesosMutex.Lock()
	defer globals.MapaProcesosMutex.Unlock()
	general.LogLockeo("Mapa Procesos", "desalojarYEnviarProceso")
	defer general.LogUnlockeo("Mapa Procesos", "desalojarYEnviarProceso")

	ipCPU, puertoCPU, nombreCPU, ok := general.BuscarCpuPorPID(pidEnExec)
	slog.Debug(fmt.Sprint("SRT - CPU del proceso a desalojar: ", nombreCPU))
	if ok {
		pidProcesoAEjecutar := globals.ESTADOS.READY[0]
		proceso := globals.MapaProcesos[pidProcesoAEjecutar]
		pcProcesoAEjecutar := proceso.Pcb.PC
		respuestaInterrupcion, err := general.EnviarInterrupcionACPU(ipCPU, puertoCPU, nombreCPU, pidEnExec)
		if err != nil {
			log.Fatal("Error en interrupción:", err)
		}
		general.ActualizarPC(pidEnExec, respuestaInterrupcion.PC)

		procesoDesalojado := globals.MapaProcesos[pidEnExec]

		ExecuteAReady(procesoDesalojado, "")

		// LOG Desalojo: ## (<PID>) - Desalojado por algoritmo SJF/SRT
		slog.Info(fmt.Sprintf("## (%d) - Desalojado por algoritmo SJF/SRT", procesoDesalojado.Pcb.Pid))

		general.EnviarProcesoAEjecutar_ACPU(ipCPU, puertoCPU, pidProcesoAEjecutar, pcProcesoAEjecutar, nombreCPU)

		aExecute(proceso)

	} else {
		slog.Debug(fmt.Sprintf("No se encontró la CPU que ejecuta el PID %d al momento de desalojar", pidEnExec))
	}
	slog.Debug(fmt.Sprint("Notificado Replanificar SRT"))
}

func ExecuteAReady(proceso globals.Proceso, razon string) {
	proceso = general.ActualizarMetricas(proceso, proceso.Estado_Actual)

	proceso.Estado_Actual = globals.READY
	general.LogIntentoLockeo("Mapa Procesos", "ExecuteAReady")
	globals.MapaProcesosMutex.Lock()
	general.LogLockeo("Mapa Procesos", "ExecuteAReady")
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso
	globals.MapaProcesosMutex.Unlock()
	general.LogUnlockeo("Mapa Procesos", "ExecuteAReady")

	//slog.Debug("Se quiere lockear en ExecuteAReady")
	globals.EstadosMutex.Lock()
	//slog.Debug("Se lockear en ExecuteAReady")
	ok := EliminarProcesoDeCola(&globals.ESTADOS.EXECUTE, proceso.Pcb.Pid)
	if !ok {
		slog.Debug(fmt.Sprintf("PID %d no se encontro en EXECUTE en ExecuteAReady", proceso.Pcb.Pid))
		return
	}

	ok = AgregarProcesoACola(&globals.ESTADOS.READY, proceso.Pcb.Pid)
	if !ok {
		slog.Debug(fmt.Sprintf("PID %d ya estaba en READY, se evitó duplicación", proceso.Pcb.Pid))
	}
	//slog.Debug("Se quiere deslockear en ExecuteAReady")
	globals.EstadosMutex.Unlock()
	//slog.Debug("Se deslockear en ExecuteAReady")
	// LOG Cambio de Estado: ## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado EXECUTE al estado READY", proceso.Pcb.Pid))
}
