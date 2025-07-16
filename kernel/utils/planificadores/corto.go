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

		globals.MapaProcesosMutex.Lock()
		globals.EstadosMutex.Lock()

		ejecutarUnProceso()

		globals.EstadosMutex.Unlock()
		globals.MapaProcesosMutex.Unlock()
	}
}

func planificadorSJF() {
	for {
		general.Wait(globals.Sem_Cpus)
		general.Wait(globals.Sem_ProcesosEnReady)

		globals.MapaProcesosMutex.Lock()
		globals.EstadosMutex.Lock()

		ordenarReadyPorRafaga()
		ejecutarUnProceso()

		globals.EstadosMutex.Unlock()
		globals.MapaProcesosMutex.Unlock()
	}
}

func planificadorSRT() {
	for {
		<-globals.SrtReplanificarChan

		// Chequeo los 4 posibles casos

		general.LogIntentoLockeo("MapaProcesos", "planificadorSRT")
		globals.MapaProcesosMutex.Lock()
		general.LogLockeo("MapaProcesos", "planificadorSRT")
		general.LogIntentoLockeo("Estados", "planificadorSRT")
		globals.EstadosMutex.Lock()
		general.LogLockeo("Estados", "planificadorSRT")

		if hayProcesosEnReady() && hayCpusLibres() {

			slog.Debug(fmt.Sprint("SRT - Hay procesos en ready y hay cpus libres"))

			ordenarReadyPorRafaga()
			ejecutarUnProceso()

		}

		if hayProcesosEnReady() && !hayCpusLibres() {
			// Caso desalojo

			slog.Debug(fmt.Sprint("SRT - Hay procesos en ready y NO hay cpus libres"))

			pidEnExec, hayQueDesalojar := verificarDesalojo()
			if hayQueDesalojar {
				slog.Debug(fmt.Sprint("SRT - PID elegido a deslojar: ", pidEnExec))
				desalojarYEnviarProceso(pidEnExec)
			}
		}

		if !hayProcesosEnReady() && hayCpusLibres() {
			slog.Debug(fmt.Sprint("SRT - No hay procesos en ready y hay cpus libres. No se hace nada"))
			// No hacemos nada
		}

		if !hayProcesosEnReady() && !hayCpusLibres() {
			slog.Debug(fmt.Sprint("SRT - No hay ni procesos en ready ni cpus libres. No se hace nada"))
			// No hacemos nada
		}

		/*
			slog.Debug(fmt.Sprint("COLAS:"))
			slog.Debug(fmt.Sprint("NEW:", globals.ESTADOS.NEW))
			slog.Debug(fmt.Sprint("READY:", globals.ESTADOS.READY))
			slog.Debug(fmt.Sprint("EXECUTE:", globals.ESTADOS.EXECUTE))
			slog.Debug(fmt.Sprint("BLOCKED:", globals.ESTADOS.BLOCKED))
			slog.Debug(fmt.Sprint("SUSP_READY:", globals.ESTADOS.SUSP_READY))
			slog.Debug(fmt.Sprint("SUSP_BLOCKED:", globals.ESTADOS.SUSP_BLOCKED))
		*/

		globals.EstadosMutex.Unlock()
		general.LogUnlockeo("Estados", "planificadorSRT")
		globals.MapaProcesosMutex.Unlock()
		general.LogUnlockeo("MapaProcesos", "planificadorSRT")

	}
}

func ActualizarEstimado(pid int64, rafagaReal float64) {
	// Me imagino que esto se usa cuando se termina de ejecutar un proceso

	slog.Debug(fmt.Sprintf("Actualizando estimado: %d", pid))
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

func elegirCPUlibre() (string, int64, string, bool) {
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
		return cpu.Handshake.IP, cpu.Handshake.Puerto, cpu.Handshake.Nombre, true
	} else {
		// Si devuelve esto hay un error, porque esta funcion se tiene que ejecutar cuando el semaforo lo permita
		log.Println("No se encontro CPU libre")
		return "", -1, "", false
	}
}

func hayProcesosEnReady() bool {
	return len(globals.ESTADOS.READY) > 0
}

func hayCpusLibres() bool {
	globals.ListaCPUsMutex.Lock()
	defer globals.ListaCPUsMutex.Unlock()
	for _, cpu := range globals.ListaCPUs {
		//slog.Debug(fmt.Sprintf("CPU %s, estado: %t, PID: %d", cpu.Handshake.Nombre, cpu.EstaLibre, cpu.PIDActual))
		if cpu.EstaLibre {
			return true
		}
	}
	return false
}

// Chequea si hay que desalojar. Si hay que desalojar, devuelve el PID que esta ejecutando
func verificarDesalojo() (int64, bool) {

	ordenarReadyPorRafaga()
	pidEnExec, restanteExec := buscarProcesoEnExecuteDeMenorRafagaRestante()
	rafagaNuevo := globals.MapaProcesos[globals.ESTADOS.READY[0]].Rafaga.Est_Sgte

	slog.Debug(fmt.Sprintf("VerificarDesalojo: rafagaNuevo (%f) - restanteExec (%f)", rafagaNuevo, restanteExec))

	if rafagaNuevo < restanteExec {
		return pidEnExec, true
	}

	return -1, false

}

func buscarProcesoEnExecuteDeMenorRafagaRestante() (int64, float64) {
	slog.Debug(fmt.Sprint("Buscando menor rafaga restante en EXECUTE: ", globals.ESTADOS.EXECUTE))

	var pidMenorRafaga int64
	pidMenorRafaga = globals.ESTADOS.EXECUTE[0]
	menorRafagaRestante := rafagaRestante(globals.ESTADOS.EXECUTE[0])
	for i := range globals.ESTADOS.EXECUTE {
		// Si la posicion i esta libre
		pidActual := globals.ESTADOS.EXECUTE[i]
		rafagaRestanteActual := rafagaRestante(pidActual)
		slog.Debug(fmt.Sprintf(" - PID %d: %f", pidMenorRafaga, menorRafagaRestante))
		if rafagaRestanteActual < menorRafagaRestante {
			pidMenorRafaga = pidActual
			menorRafagaRestante = rafagaRestanteActual
		}
	}

	slog.Debug(fmt.Sprint(" --- Elegido para desalojar: ", pidMenorRafaga, menorRafagaRestante))
	return pidMenorRafaga, menorRafagaRestante
}

func rafagaRestante(pid int64) float64 {

	//slog.Debug(fmt.Sprint("PID: ", pid))
	proceso := globals.MapaProcesos[pid]
	//slog.Debug(fmt.Sprint("Proceso: ", proceso))
	rafaga := proceso.Rafaga.Est_Sgte
	//slog.Debug(fmt.Sprint("Rafaga: ", rafaga))                                       // float64
	tiempoPasado := float64(time.Since(proceso.UltimoCambioDeEstado).Milliseconds()) // tiempo en ms
	//slog.Debug(fmt.Sprint("Tiempo pasado: ", tiempoPasado))

	slog.Debug(fmt.Sprintf("PID %d - Rafaga restante: %f", pid, rafaga-tiempoPasado))
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

	procesoAEjecutar := globals.ESTADOS.READY[0]
	ip, port, nombre, hayCPU := elegirCPUlibre()
	if hayCPU {
		proceso := globals.MapaProcesos[procesoAEjecutar]
		general.EnviarProcesoAEjecutar_ACPU(ip, port, proceso.Pcb.Pid, proceso.Pcb.PC, nombre)
		CambiarEstado(procesoAEjecutar, globals.READY, globals.EXECUTE)
	}

}

func desalojarYEnviarProceso(pidEnExec int64) {

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

		CambiarEstado(procesoDesalojado.Pcb.Pid, globals.EXECUTE, globals.READY)

		// LOG Desalojo: ## (<PID>) - Desalojado por algoritmo SJF/SRT
		slog.Info(fmt.Sprintf("## (%d) - Desalojado por algoritmo SJF/SRT", procesoDesalojado.Pcb.Pid))

		general.EnviarProcesoAEjecutar_ACPU(ipCPU, puertoCPU, pidProcesoAEjecutar, pcProcesoAEjecutar, nombreCPU)

		CambiarEstado(pidProcesoAEjecutar, globals.READY, globals.EXECUTE)

	} else {
		slog.Debug(fmt.Sprintf("No se encontró la CPU que ejecuta el PID %d al momento de desalojar", pidEnExec))
	}
	//slog.Debug(fmt.Sprint("Notificado Replanificar SRT"))
}
