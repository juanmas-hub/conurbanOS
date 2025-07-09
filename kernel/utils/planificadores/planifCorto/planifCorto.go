package utils_planifCorto

import (
	"log"

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

		//log.Print("Se quiere bloquear en planificadorFIFO")
		globals.EstadosMutex.Lock()
		log.Print("Se bloqueo en planificadorFIFO")

		ejecutarUnProceso()

		//log.Print("Se quiere desbloquear en planificadorFIFO")
		globals.EstadosMutex.Unlock()
		log.Print("Se desbloqueo en planificadorFIFO")
	}
}

func planificadorSJF() {
	for {
		general.Wait(globals.Sem_Cpus)
		general.Wait(globals.Sem_ProcesosEnReady)

		globals.EstadosMutex.Lock()
		log.Print("Se loqueo en planificadorSJF")

		ordenarReadyPorRafaga()
		ejecutarUnProceso()

		globals.EstadosMutex.Unlock()
		log.Print("Se desloqueo en planificadorSJF")
	}
}

func planificadorSRT() {
	for {
		<-globals.SrtReplanificarChan

		// Chequeo los 4 posibles casos

		if hayProcesosEnReady() && hayCpusLibres() {
			globals.EstadosMutex.Lock()
			log.Print("Se loqueo en planificadorSRT")

			ordenarReadyPorRafaga()
			ejecutarUnProceso()

			globals.EstadosMutex.Unlock()
			log.Print("Se desloqueo en planificadorSRT")
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

func ActualizarEstimado(pid int64, rafagaReal int64) {
	// Me imagino que esto se usa cuando se termina de ejecutar un proceso

	globals.MapaProcesosMutex.Lock()
	proceso := globals.MapaProcesos[pid]
	globals.MapaProcesosMutex.Unlock()

	log.Print(proceso.Rafaga)

	alpha := globals.KernelConfig.Alpha
	est_ant := proceso.Rafaga.Est_Sgte

	proceso.Rafaga.Est_Ant = est_ant
	proceso.Rafaga.Raf_Ant = rafagaReal
	proceso.Rafaga.Est_Sgte = rafagaReal*alpha + est_ant*(1-alpha)
	// Est(n+1) =  R(n) + (1-) Est(n) ;    [0,1]

	globals.MapaProcesosMutex.Lock()
	globals.MapaProcesos[pid] = proceso
	globals.MapaProcesosMutex.Unlock()
}
