package utils_planifCorto

import (
	"fmt"
	"log"
	"log/slog"
	"sort"
	"time"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
	general "github.com/sisoputnfrba/tp-golang/kernel/utils/general"
)

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

	//log.Printf("Proceso agregado a EXEC. Ahora tiene %d procesos", len(globals.ESTADOS.EXECUTE))

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
	log.Print("Se loqueo en buscarProcesoEnExecute")
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

func buscarProcesoEnExecuteDeMenorRafagaRestante() (int64, int64) {
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

func rafagaRestante(pid int64) int64 {
	return globals.MapaProcesos[pid].Rafaga.Est_Sgte - int64(time.Now().Sub(globals.MapaProcesos[pid].UltimoCambioDeEstado))
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
}

func ejecutarUnProceso() {
	//log.Print("Se quiere loquear MapaProcesos en ejecutarUnProceso")
	procesoAEjecutar := globals.ESTADOS.READY[0]
	ip, port, nombre := elegirCPUlibre()
	globals.MapaProcesosMutex.Lock()
	proceso := globals.MapaProcesos[procesoAEjecutar]
	globals.MapaProcesosMutex.Unlock()
	general.EnviarProcesoAEjecutar_ACPU(ip, port, proceso.Pcb.Pid, proceso.Pcb.PC, nombre)
	aExecute(proceso)
	//log.Print("Se unloquea MapaProcesos en ejecutarUnProceso")
}

func desalojarYEnviarProceso(pidEnExec int64) {
	ipCPU, puertoCPU, nombreCPU, ok := general.BuscarCpuPorPID(pidEnExec)
	if ok {
		globals.EstadosMutex.Lock()
		log.Print("Se loqueo en desalojarYEnviarProceso")
		globals.MapaProcesosMutex.Lock()
		pidProcesoAEjecutar := globals.ESTADOS.READY[0]
		proceso := globals.MapaProcesos[pidProcesoAEjecutar]
		pcProcesoAEjecutar := proceso.Pcb.PC
		globals.MapaProcesosMutex.Unlock()
		globals.EstadosMutex.Unlock()
		log.Print("Se desloqueo en desalojarYEnviarProceso")

		respuestaInterrupcion, err := general.EnviarInterrupcionACPU(ipCPU, puertoCPU, nombreCPU, pidEnExec)
		if err != nil {
			log.Fatal("Error en interrupción:", err)
		}
		general.ActualizarPC(pidEnExec, respuestaInterrupcion.PC)
		general.EnviarProcesoAEjecutar_ACPU(ipCPU, puertoCPU, pidProcesoAEjecutar, pcProcesoAEjecutar, nombreCPU)
		log.Printf("Se desalojo el proceso %d", pidEnExec)

		// LOG Desalojo: ## (<PID>) - Desalojado por algoritmo SJF/SRT
		slog.Info(fmt.Sprintf("## (%d) - Desalojado por algoritmo SJF/SRT", proceso.Pcb.Pid))

		globals.MapaProcesosMutex.Lock()
		aExecute(proceso)

		procesoDesalojado := globals.MapaProcesos[pidEnExec]
		globals.MapaProcesosMutex.Unlock()

		ExecuteAReady(procesoDesalojado, "")

	} else {
		log.Printf("No se encontró la CPU que ejecuta el PID %d", pidEnExec)
	}
}

func ExecuteAReady(proceso globals.Proceso, razon string) {
	proceso = general.ActualizarMetricas(proceso, proceso.Estado_Actual)
	ahora := time.Now()
	tiempoEnEstado := ahora.Sub(proceso.UltimoCambioDeEstado)
	ActualizarEstimado(proceso.Pcb.Pid, int64(tiempoEnEstado))

	proceso.Estado_Actual = globals.READY
	globals.MapaProcesosMutex.Lock()
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso
	globals.MapaProcesosMutex.Unlock()

	//log.Print("Se quiere bloquear en ExecuteABlocked")
	globals.EstadosMutex.Lock()
	log.Print("Se bloqueo en ExecuteABlocked")
	pos := buscarProcesoEnExecute(proceso.Pcb.Pid)
	globals.ESTADOS.EXECUTE = append(globals.ESTADOS.EXECUTE[:pos], globals.ESTADOS.EXECUTE[pos+1:]...)
	globals.ESTADOS.READY = append(globals.ESTADOS.READY, proceso.Pcb.Pid)
	//log.Print("Se quiere desbloquear en ExecuteABlocked")
	globals.EstadosMutex.Unlock()
	log.Print("Se desbloqueo en ExecuteABlocked")

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
