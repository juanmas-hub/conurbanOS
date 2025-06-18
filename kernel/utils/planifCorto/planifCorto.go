package utils_planifCorto

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"sort"
	"time"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
	general "github.com/sisoputnfrba/tp-golang/kernel/utils/general"
)

// ----- FUNCIONES EXPORTADAS -------

func EjecutarPlanificadorCortoPlazo() {
	// Es un while infinito, pero se queda esperando al principio a que haya CPUs libres (wait es bloqueante)
	// y que haya procesos en ready

	if globals.KernelConfig.Scheduler_algorithm == "FIFO" {
		for {
			general.Wait(globals.Sem_Cpus)            // Espero a que haya Cpus libres
			general.Wait(globals.Sem_ProcesosEnReady) // Espero a que haya procesos en Ready

			globals.EstadosMutex.Lock()

			ejecutarUnProceso()

			globals.EstadosMutex.Unlock()
		}
	}

	// Basicamente lo mismo que FIFO, pero reordenando la cola por rafaga
	if globals.KernelConfig.Scheduler_algorithm == "SJF" {
		for {
			general.Wait(globals.Sem_Cpus)
			general.Wait(globals.Sem_ProcesosEnReady)

			globals.EstadosMutex.Lock()

			ordenarReadyPorRafaga()
			ejecutarUnProceso()

			globals.EstadosMutex.Unlock()
		}
	}

	if globals.KernelConfig.Scheduler_algorithm == "SRT" {

		// No uso los semaforos de CPU ni de procesos en Ready
		go planificadorSRT()

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

func desalojarYEnviarProceso(pidEnExec int64) {
	ipCPU, puertoCPU, ok := general.BuscarCpuPorPID(pidEnExec)
	if ok {
		globals.EstadosMutex.Lock()
		globals.MapaProcesosMutex.Lock()
		pidProcesoAEjecutar := globals.ESTADOS.READY[0]
		proceso := globals.MapaProcesos[pidProcesoAEjecutar]
		pcProcesoAEjecutar := proceso.Pcb.PC
		globals.MapaProcesosMutex.Unlock()
		globals.EstadosMutex.Unlock()

		respuestaInterrupcion, err := general.EnviarInterrupcionACPU(ipCPU, puertoCPU, pidEnExec)
		if err != nil {
			log.Fatal("Error en interrupción:", err)
		}
		general.ActualizarPC(pidEnExec, respuestaInterrupcion.PC)
		general.EnviarProcesoAEjecutar_ACPU(ipCPU, puertoCPU, pidProcesoAEjecutar, pcProcesoAEjecutar)
		log.Printf("Se desalojo el proceso %d", pidEnExec)

		// LOG Desalojo: ## (<PID>) - Desalojado por algoritmo SJF/SRT
		slog.Info(fmt.Sprintf("## (%d) - Desalojado por algoritmo SJF/SRT", proceso.Pcb.Pid))

		globals.MapaProcesosMutex.Lock()
		aExecute(proceso)
		globals.MapaProcesosMutex.Unlock()

	} else {
		log.Printf("No se encontró la CPU que ejecuta el PID %d", pidEnExec)
	}
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
	globals.MapaProcesosMutex.Lock()
	procesoAEjecutar := globals.ESTADOS.READY[0]
	log.Print("Proceso a ejecutar: ", procesoAEjecutar)
	ip, port := elegirCPUlibre()
	proceso := globals.MapaProcesos[procesoAEjecutar]
	general.EnviarProcesoAEjecutar_ACPU(ip, port, proceso.Pcb.Pid, proceso.Pcb.PC)
	aExecute(proceso)
	globals.MapaProcesosMutex.Unlock()
	//log.Print("Se unloquea MapaProcesos en ejecutarUnProceso")
}

// Capaz esta funcion no hace falta - hay que ver si las devoluciones de CPU son unicamente syscalls
// o hay mas casos.
// Si son solo syscalls, esta funcion es al pedo
func DevolucionProceso(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var devolucion globals.DevolucionProceso
	err := decoder.Decode(&devolucion)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	// Si se bloquea, lo hace por Syscall IO, aca no tenemos en cuenta eso

	go func() {

		if devolucion.Motivo == globals.REPLANIFICAR_PROCESO {
			// Replanificaciones:

			general.ActualizarPC(devolucion.Pid, devolucion.PC)

			// Se selecciona el proximo proceso a ejecutar
			// La CPU queda esperando?? PAGINA 13

		}

		if devolucion.Motivo == globals.FIN_PROCESO {
			general.FinalizarProceso(devolucion.Pid)
			general.LiberarCPU(devolucion.Nombre_CPU)
		}

	}()

	log.Println("Se devolvió un proceso")
	log.Printf("%+v\n", devolucion)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func ActualizarEstimado(pid int64, rafagaReal int64) {
	// Me imagino que esto se usa cuando se termina de ejecutar un proceso

	proceso := globals.MapaProcesos[pid]
	alpha := globals.KernelConfig.Alpha
	ant := proceso.Rafaga.Est_Sgte
	est_ant := proceso.Rafaga.Est_Sgte

	proceso.Rafaga.Raf_Ant = ant
	proceso.Rafaga.Est_Ant = est_ant
	proceso.Rafaga.Est_Sgte = rafagaReal*alpha + ant*(1-alpha)
	// Est(n+1) =  R(n) + (1-) Est(n) ;    [0,1]

	globals.MapaProcesos[pid] = proceso
}

// ----- FUNCIONES LOCALES -------

func elegirCPUlibre() (string, int64) {
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
		return cpu.Handshake.IP, cpu.Handshake.Puerto
	} else {
		// Si devuelve esto hay un error, porque esta funcion se tiene que ejecutar cuando el semaforo lo permita
		log.Println("No se encontro CPU libre")
		return "", -1
	}
}

// Se llama con MapaProcesosMutex lockeado
func aExecute(proceso globals.Proceso) {
	// Esto funcionaría para FIFO y SJF. Nose si SRT

	estado_anterior := proceso.Estado_Actual

	proceso = general.ActualizarMetricas(proceso, proceso.Estado_Actual)
	proceso.Estado_Actual = globals.EXECUTE
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso
	globals.ESTADOS.READY = globals.ESTADOS.READY[1:]
	globals.ESTADOS.EXECUTE = append(globals.ESTADOS.EXECUTE, proceso.Pcb.Pid)

	log.Printf("Proceso agregado a EXEC. Ahora tiene %d procesos", len(globals.ESTADOS.EXECUTE))

	// LOG Cambio de Estado: ## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado %s al estado EXECUTE", proceso.Pcb.Pid, estado_anterior))
}
