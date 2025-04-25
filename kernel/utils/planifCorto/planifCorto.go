package utils_planifCorto

import (
	"log"
	"sort"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
	general "github.com/sisoputnfrba/tp-golang/kernel/utils/general"
)

func EjecutarPlanificadorCortoPlazo() {

	if globals.KernelConfig.Scheduler_algorithm == "FIFO" {
		globals.EstadosMutex.Lock()

		procesoAEjecutar := globals.ESTADOS.READY[0]
		ip, port := ElegirCPUlibre()
		general.EnviarProcesoAEjecutar_ACPU(ip, port, procesoAEjecutar)

		globals.MapaProcesosMutex.Lock()

		ReadyAExecute(globals.MapaProcesos[procesoAEjecutar])
		log.Printf("Proceso agregado a EXEC. Ahora tiene %d procesos", len(globals.ESTADOS.EXECUTE))

		globals.EstadosMutex.Unlock()
		globals.MapaProcesosMutex.Unlock()
	}

	if globals.KernelConfig.Scheduler_algorithm == "SJF" {
		globals.EstadosMutex.Lock()
		// SJF SIN DESALOJO (Se elige al proceso que tenga la rafaga estimada mas corta)
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

		procesoAEjecutar := globals.ESTADOS.READY[0]
		ip, port := ElegirCPUlibre()
		general.EnviarProcesoAEjecutar_ACPU(ip, port, procesoAEjecutar)
		globals.MapaProcesosMutex.Lock()
		ReadyAExecute(globals.MapaProcesos[procesoAEjecutar])
		log.Printf("Proceso agregado a EXEC. Ahora tiene %d procesos", len(globals.ESTADOS.EXECUTE))
		globals.EstadosMutex.Unlock()
		globals.MapaProcesosMutex.Unlock()
	}

	if globals.KernelConfig.Scheduler_algorithm == "SRT" {
		// Con desalojo
		// No se como sería esto. Capaz hay q hacer una funcion aparte porque se llamaria en momentos distintos

		if len(globals.ESTADOS.EXECUTE) > 0 {
			pidEnExec := globals.ESTADOS.EXECUTE[0]
			rafagaExec := globals.MapaProcesos[pidEnExec].Rafaga.Est_Sgte
			rafagaNuevo := globals.MapaProcesos[globals.ESTADOS.READY[0]].Rafaga.Est_Sgte

			if rafagaNuevo < rafagaExec {
				// InterrupcionACpu(pidEnExec)
			}
		}

	}
}

func ActualizarEstimado(pid int64, rafagaReal int64) {
	// En desarrollo
	//proceso := globals.MapaProcesos[pid]
	//alpha := globals.KernelConfig.Alpha
	//ant := proceso.Rafaga.Est_Sgte

	// Est(n+1) =  R(n) + (1-) Est(n) ;    [0,1]

	//globals.MapaProcesos[pid] = proceso
}

func ElegirCPUlibre() (string, int64) {
	// Hay que hacerlo. Seguramente haya que cambiar HandshakesCPU para indicar cual esta libre

	return globals.HandshakesCPU[0].IP, globals.HandshakesCPU[0].Puerto
}

func ReadyAExecute(proceso globals.Proceso) {
	// Esto funcionaría para FIFO y SJF. Nose si SRT

	proceso.Estado_Actual = globals.EXECUTE
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso
	globals.ESTADOS.READY = globals.ESTADOS.READY[1:]
	globals.ESTADOS.EXECUTE = append(globals.ESTADOS.EXECUTE, proceso.Pcb.Pid)
}
