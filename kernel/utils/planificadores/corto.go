package planificadores

import (
	"fmt"
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

		ejecutarUnProcesoFifo()
	}
}

func planificadorSJF() {
	for {
		general.Wait(globals.Sem_Cpus)
		general.Wait(globals.Sem_ProcesosEnReady)

		ejecutarUnProcesoSjf()
	}
}

func planificadorSRT() {
	for {
		<-globals.SrtReplanificarChan

		if hayProcesosEnReady() && hayCpusLibres() {

			slog.Debug(fmt.Sprint("SRT - Hay procesos en ready y hay cpus libres"))

			ejecutarUnProcesoSjf()

		} else if hayProcesosEnReady() && !hayCpusLibres() {
			// Caso desalojo

			slog.Debug(fmt.Sprint("SRT - Hay procesos en ready y NO hay cpus libres"))

			pidEnExec, hayQueDesalojar := verificarDesalojo()
			if hayQueDesalojar {
				slog.Debug(fmt.Sprint("SRT - PID elegido a deslojar: ", pidEnExec))
				desalojarYEnviarProceso(pidEnExec)
			}
		} else if !hayProcesosEnReady() && hayCpusLibres() {
			slog.Debug(fmt.Sprint("SRT - No hay procesos en ready y hay cpus libres. No se hace nada"))
			// No hacemos nada
		} else if !hayProcesosEnReady() && !hayCpusLibres() {
			slog.Debug(fmt.Sprint("SRT - No hay ni procesos en ready ni cpus libres. No se hace nada"))
			// No hacemos nada
		}

	}
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
		slog.Debug(fmt.Sprintf("No se encontro CPU libre"))
		return "", -1, "", false
	}
}

func hayProcesosEnReady() bool {
	return len(globals.Cola_ready) > 0
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
	rafagaNuevo := globals.MapaProcesos[globals.Cola_ready[0]].Rafaga.Est_Sgte

	slog.Debug(fmt.Sprintf("VerificarDesalojo: rafagaNuevo (%f) - restanteExec (%f)", rafagaNuevo, restanteExec))

	if rafagaNuevo < restanteExec {
		return pidEnExec, true
	}

	return -1, false

}

func buscarProcesoEnExecuteDeMenorRafagaRestante() (int64, float64) {
	slog.Debug(fmt.Sprint("Buscando menor rafaga restante en EXECUTE: ", globals.Cola_execute))

	var pidMenorRafaga int64
	pidMenorRafaga = globals.Cola_execute[0]
	menorRafagaRestante := rafagaRestante(globals.Cola_execute[0])
	for i := range globals.Cola_execute {
		// Si la posicion i esta libre
		pidActual := globals.Cola_execute[i]
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
	sort.Slice(globals.Cola_ready, func(i, j int) bool {
		pidI := globals.Cola_ready[i]
		pidJ := globals.Cola_ready[j]

		// De cada par de procesos sacamos la rafaga que tiene cada uno
		rafagaI := globals.MapaProcesos[pidI].Rafaga
		rafagaJ := globals.MapaProcesos[pidJ].Rafaga
		// Si la rafagaI es menor, la ponemos antes
		return rafagaI.Est_Sgte < rafagaJ.Est_Sgte
	})

	var rafagasReady []float64
	for i := range globals.Cola_ready {
		// Si la posicion i esta libre
		pidActual := globals.Cola_ready[i]
		rafagasReady = append(rafagasReady, globals.MapaProcesos[pidActual].Rafaga.Est_Sgte)
	}
	slog.Info(fmt.Sprintf("Ready ordenado por rafaga: %d, %f", globals.Cola_ready, rafagasReady))

}

func ejecutarUnProcesoFifo() {

	ip, port, nombre, hayCPU := elegirCPUlibre()
	if !hayCPU {
		return
	}

	globals.ReadyMutex.Lock()
	pid := globals.Cola_ready[0]
	globals.ReadyMutex.Unlock()

	globals.ProcesosMutex[pid].Lock()
	proceso, existe := globals.MapaProcesos[pid]
	if !existe {
		return
	}
	globals.ProcesosMutex[pid].Unlock()

	Enviar_proceso_a_cpu(ip, port, pid, proceso.Pcb.PC, nombre)
	ready_a_execute(pid)

}

func ejecutarUnProcesoSjf() {

	ip, port, nombre, hayCPU := elegirCPUlibre()
	if !hayCPU {
		return
	}

	globals.ReadyMutex.Lock()
	ordenarReadyPorRafaga()
	pid := globals.Cola_ready[0]
	globals.ReadyMutex.Unlock()

	globals.ProcesosMutex[pid].Lock()
	proceso, existe := globals.MapaProcesos[pid]
	if !existe {
		return
	}
	globals.ProcesosMutex[pid].Unlock()

	Enviar_proceso_a_cpu(ip, port, pid, proceso.Pcb.PC, nombre)
	ready_a_execute(pid)

}

/*
func desalojarYEnviarProceso(pidEnExec int64) {

	ipCPU, puertoCPU, nombreCPU, ok := general.BuscarCpuPorPID(pidEnExec)
	slog.Debug(fmt.Sprint("SRT - CPU del proceso a desalojar: ", nombreCPU))
	if ok {
		pidProcesoAEjecutar := globals.Cola_ready[0]
		procesoAEjecutar := globals.MapaProcesos[pidProcesoAEjecutar]
		pcProcesoAEjecutar := procesoAEjecutar.Pcb.PC

		procesoEnExec, existe := globals.MapaProcesos[pidEnExec]
		if !existe {
			slog.Debug(fmt.Sprint("El proceso ya no existe, probablemente finalizo"))
			general.EnviarProcesoAEjecutar_ACPU(ipCPU, puertoCPU, pidProcesoAEjecutar, pcProcesoAEjecutar, nombreCPU)
			CambiarEstado(pidProcesoAEjecutar, globals.READY, globals.EXECUTE)
			return
		}

		if procesoEnExec.Estado_Actual != globals.EXECUTE {
			slog.Debug(fmt.Sprint("El proceso ya no esta en EXECUTE, probablemente solicito Syscall"))
			general.EnviarProcesoAEjecutar_ACPU(ipCPU, puertoCPU, pidProcesoAEjecutar, pcProcesoAEjecutar, nombreCPU)
			CambiarEstado(pidProcesoAEjecutar, globals.READY, globals.EXECUTE)
			return
		}

		respuestaInterrupcion, err := general.EnviarInterrupcionACPU(ipCPU, puertoCPU, nombreCPU, pidEnExec)
		if err != nil {
			slog.Debug(fmt.Sprint("Error en interrupción:", err))
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
}*/

func desalojarYEnviarProceso(pidEnExec int64) {

	ipCPU, puertoCPU, nombreCPU, ok := general.BuscarCpuPorPID(pidEnExec)
	slog.Debug(fmt.Sprint("SRT - CPU del proceso a desalojar: ", nombreCPU))
	if ok {

		globals.ReadyMutex.Lock()

		pid_a_ejecutar := globals.Cola_ready[0]
		globals.ProcesosMutex[pid_a_ejecutar].Lock()
		pc_a_ejecutar := globals.MapaProcesos[pid_a_ejecutar].Pcb.PC
		globals.ProcesosMutex[pid_a_ejecutar].Unlock()

		globals.ReadyMutex.Unlock()

		globals.ExecuteMutex.Lock()
		globals.ProcesosMutex[pidEnExec].Lock()

		proceso_en_exec, existe := globals.MapaProcesos[pidEnExec]
		if !existe || proceso_en_exec.Estado_Actual != globals.EXECUTE {

			Enviar_proceso_a_cpu(ipCPU, puertoCPU, pid_a_ejecutar, pc_a_ejecutar, nombreCPU)
			ready_a_execute(pid_a_ejecutar)

			return
		}

		globals.ProcesosMutex[pidEnExec].Unlock()
		globals.ExecuteMutex.Unlock()

		respuestaInterrupcion, err := enviar_interrupcion_a_cpu(ipCPU, puertoCPU, nombreCPU, pidEnExec)
		if err != nil {
			slog.Debug(fmt.Sprint("Error en interrupción:", err))
		}

		// LOG Desalojo: ## (<PID>) - Desalojado por algoritmo SJF/SRT
		slog.Info(fmt.Sprintf("## (%d) - Desalojado por algoritmo SJF/SRT", pidEnExec))

		globals.ProcesosMutex[pid_a_ejecutar].Lock()
		general.ActualizarPC(pidEnExec, respuestaInterrupcion.PC)
		globals.ProcesosMutex[pid_a_ejecutar].Unlock()

		execute_a_ready(pidEnExec)
		Enviar_proceso_a_cpu(ipCPU, puertoCPU, pid_a_ejecutar, pc_a_ejecutar, nombreCPU)
		ready_a_execute(pid_a_ejecutar)

	} else {
		slog.Debug(fmt.Sprintf("No se encontró la CPU que ejecuta el PID %d al momento de desalojar", pidEnExec))
	}
	//slog.Debug(fmt.Sprint("Notificado Replanificar SRT"))
}
