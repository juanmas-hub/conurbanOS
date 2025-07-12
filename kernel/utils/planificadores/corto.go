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

		globals.EstadosMutex.Lock()

		slog.Debug(fmt.Sprintf(" ---- Algoritmo Corto Plazo ----"))
		ejecutarUnProceso()

		globals.EstadosMutex.Unlock()
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
		<-globals.SrtReplanificarChan

		// Chequeo los 4 posibles casos

		if hayProcesosEnReady() && hayCpusLibres() {
			globals.EstadosMutex.Lock()

			ordenarReadyPorRafaga()
			ejecutarUnProceso()

			globals.EstadosMutex.Unlock()
		}

		if hayProcesosEnReady() && !hayCpusLibres() {
			// Caso desalojo
			pidEnExec, hayQueDesalojar := verificarDesalojo()
			if hayQueDesalojar {
				desalojarYEnviarProceso(pidEnExec)
			}
		}

		if !hayProcesosEnReady() && hayCpusLibres() {
			// No hacemos nada
		}

		if !hayProcesosEnReady() && !hayCpusLibres() {
			// No hacemos nada
		}
	}
}

func ActualizarEstimado(pid int64, rafagaReal float64) {
	// Me imagino que esto se usa cuando se termina de ejecutar un proceso

	slog.Debug(fmt.Sprintf("Rafaga real: %f", rafagaReal))

	globals.MapaProcesosMutex.Lock()
	proceso := globals.MapaProcesos[pid]
	globals.MapaProcesosMutex.Unlock()

	alpha := globals.KernelConfig.Alpha
	est_ant := proceso.Rafaga.Est_Sgte

	proceso.Rafaga.Est_Ant = est_ant
	proceso.Rafaga.Raf_Ant = rafagaReal
	proceso.Rafaga.Est_Sgte = rafagaReal*alpha + est_ant*(1-alpha)
	// Est(n+1) =  R(n) + (1-) Est(n) ;    [0,1]

	slog.Debug(fmt.Sprintf("Rafaga actualizada de PID %d: %f", proceso.Pcb.Pid, proceso.Rafaga))

	globals.MapaProcesosMutex.Lock()
	globals.MapaProcesos[pid] = proceso
	globals.MapaProcesosMutex.Unlock()
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

// Se llama con MapaProcesosMutex lockeado
func aExecute(proceso globals.Proceso) {
	// Esto funcionaría para FIFO y SJF. Nose si SRT

	estado_anterior := proceso.Estado_Actual

	proceso = general.ActualizarMetricas(proceso, proceso.Estado_Actual)
	proceso.Estado_Actual = globals.EXECUTE
	globals.MapaProcesosMutex.Lock()
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso
	globals.MapaProcesosMutex.Unlock()
	globals.ESTADOS.READY = globals.ESTADOS.READY[1:]
	globals.ESTADOS.EXECUTE = append(globals.ESTADOS.EXECUTE, proceso.Pcb.Pid)

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

	ordenarReadyPorRafaga()
	pidEnExec, restanteExec := buscarProcesoEnExecuteDeMenorRafagaRestante()
	rafagaNuevo := globals.MapaProcesos[globals.ESTADOS.READY[0]].Rafaga.Est_Sgte

	if rafagaNuevo < restanteExec {
		return pidEnExec, true
	}

	return -1, false

}

func buscarProcesoEnExecuteDeMenorRafagaRestante() (int64, float64) {
	var pidMenorRafaga int64
	pidMenorRafaga = globals.ESTADOS.EXECUTE[0]
	for i := range globals.ESTADOS.EXECUTE {
		// Si la posicion i esta libre
		pidActual := globals.ESTADOS.EXECUTE[i]
		if rafagaRestante(pidActual) < rafagaRestante(pidMenorRafaga) {
			pidMenorRafaga = pidActual
		}
	}

	return pidMenorRafaga, rafagaRestante(pidMenorRafaga)
}

func rafagaRestante(pid int64) float64 {

	proceso := globals.MapaProcesos[pid]
	rafaga := proceso.Rafaga.Est_Sgte                                                // float64
	tiempoPasado := float64(time.Since(proceso.UltimoCambioDeEstado).Milliseconds()) // tiempo en ms

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
	globals.MapaProcesosMutex.Lock()
	proceso := globals.MapaProcesos[procesoAEjecutar]
	globals.MapaProcesosMutex.Unlock()
	general.EnviarProcesoAEjecutar_ACPU(ip, port, proceso.Pcb.Pid, proceso.Pcb.PC, nombre)
	aExecute(proceso)
}

func desalojarYEnviarProceso(pidEnExec int64) {
	ipCPU, puertoCPU, nombreCPU, ok := general.BuscarCpuPorPID(pidEnExec)
	if ok {
		globals.EstadosMutex.Lock()
		globals.MapaProcesosMutex.Lock()
		pidProcesoAEjecutar := globals.ESTADOS.READY[0]
		proceso := globals.MapaProcesos[pidProcesoAEjecutar]
		pcProcesoAEjecutar := proceso.Pcb.PC
		globals.MapaProcesosMutex.Unlock()
		globals.EstadosMutex.Unlock()

		respuestaInterrupcion, err := general.EnviarInterrupcionACPU(ipCPU, puertoCPU, nombreCPU, pidEnExec)
		if err != nil {
			log.Fatal("Error en interrupción:", err)
		}
		general.ActualizarPC(pidEnExec, respuestaInterrupcion.PC)
		general.EnviarProcesoAEjecutar_ACPU(ipCPU, puertoCPU, pidProcesoAEjecutar, pcProcesoAEjecutar, nombreCPU)

		// LOG Desalojo: ## (<PID>) - Desalojado por algoritmo SJF/SRT
		slog.Info(fmt.Sprintf("## (%d) - Desalojado por algoritmo SJF/SRT", proceso.Pcb.Pid))

		globals.MapaProcesosMutex.Lock()
		globals.EstadosMutex.Lock()
		aExecute(proceso)
		globals.EstadosMutex.Unlock()

		procesoDesalojado := globals.MapaProcesos[pidEnExec]
		globals.MapaProcesosMutex.Unlock()

		ExecuteAReady(procesoDesalojado, "")

	} else {
		log.Printf("No se encontró la CPU que ejecuta el PID %d", pidEnExec)
	}
}

func ExecuteAReady(proceso globals.Proceso, razon string) {
	ahora := time.Now()
	tiempoEnEstado := ahora.Sub(proceso.UltimoCambioDeEstado)
	proceso = general.ActualizarMetricas(proceso, proceso.Estado_Actual)
	if globals.KernelConfig.Scheduler_algorithm != "FIFO" {
		ActualizarEstimado(proceso.Pcb.Pid, float64(tiempoEnEstado.Milliseconds()))
	}

	proceso.Estado_Actual = globals.READY
	globals.MapaProcesosMutex.Lock()
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso
	globals.MapaProcesosMutex.Unlock()

	globals.EstadosMutex.Lock()
	pos := buscarProcesoEnExecute(proceso.Pcb.Pid)
	globals.ESTADOS.EXECUTE = append(globals.ESTADOS.EXECUTE[:pos], globals.ESTADOS.EXECUTE[pos+1:]...)
	globals.ESTADOS.READY = append(globals.ESTADOS.READY, proceso.Pcb.Pid)
	globals.EstadosMutex.Unlock()

	// LOG Cambio de Estado: ## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado EXECUTE al estado READY", proceso.Pcb.Pid))
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
