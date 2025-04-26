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
		globals.EstadosMutex.Lock()
		// Con desalojo
		// Primero ordenamos READY por rafaga
		sort.Slice(globals.ESTADOS.READY, func(i, j int) bool {
			pidI := globals.ESTADOS.READY[i]
			pidJ := globals.ESTADOS.READY[j]

			// De cada par de procesos sacamos la rafaga que tiene cada uno
			rafagaI := globals.MapaProcesos[pidI].Rafaga
			rafagaJ := globals.MapaProcesos[pidJ].Rafaga
			// Si la rafagaI es menor, la ponemos antes
			return rafagaI.Est_Sgte < rafagaJ.Est_Sgte
		})
		// Si hay un proceso en EXECUTE -> comparamos rafagas
		if len(globals.ESTADOS.EXECUTE) > 0 {
			pidEnExec := globals.ESTADOS.EXECUTE[0]
			rafagaExec := globals.MapaProcesos[pidEnExec].Rafaga.Est_Sgte
			rafagaNuevo := globals.MapaProcesos[globals.ESTADOS.READY[0]].Rafaga.Est_Sgte

			if rafagaNuevo < rafagaExec {
				// OjO !! Esto debe estar mal. Hay que saber cual es la CPU que queremos desalojar
				cpu := globals.HandshakesCPU[0]
				ipCPU := cpu.IP
				puertoCPU := cpu.Puerto
				general.EnviarInterrupcionACPU(ipCPU, puertoCPU, pidEnExec)
				// Aca la logica para mandar el proceso con rafaga mas corta - despues lo hago me voy a tocar
			}
		}
		// Si no hay ningun proceso en EXECUTE -> simplemente agregamos el primero de READY
		procesoAEjecutar := globals.ESTADOS.READY[0]
		ip, port := ElegirCPUlibre()
		general.EnviarProcesoAEjecutar_ACPU(ip, port, procesoAEjecutar)
		globals.MapaProcesosMutex.Lock()
		ReadyAExecute(globals.MapaProcesos[procesoAEjecutar])
		log.Printf("Proceso agregado a EXEC. Ahora tiene %d procesos", len(globals.ESTADOS.EXECUTE))
		globals.EstadosMutex.Unlock()
		globals.MapaProcesosMutex.Unlock()
	}
}

// Me imagino que esto se usa cuando se termina de ejecutar un proceso
func ActualizarEstimado(pid int64, rafagaReal int64) {

	proceso := globals.MapaProcesos[pid]
	alpha := globals.KernelConfig.Alpha
	ant := proceso.Rafaga.Est_Sgte
	proceso.Rafaga.Est_Sgte = rafagaReal*alpha + ant*(1-alpha)
	// Est(n+1) =  R(n) + (1-) Est(n) ;    [0,1]

	globals.MapaProcesos[pid] = proceso
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
