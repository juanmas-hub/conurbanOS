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

		/*
			Nuevo proceso en Ready && no hay CPUs libres =>
					- rafaga mas corta que los que estan en ejecucion => desalojar el de tiempo restante mas alto
					- no es mas corta => lo dejamos en la cola de ready

			Ejecutamos en dos hilos distintos:
				- caso no desalojo: como el resto de algoritmos, cuando haya cpus libres y procesos en ready, ejecuta
				- caso desalojo: cuando llegue un proceso en ready, se fija si tiene prioridad. En el caso que el proceso se debe
								 quedar en ready,
		*/

		go casoSRTnodesalojo()
		go casoSRTdesalojo()

	}
}

func casoSRTnodesalojo() {
	for {
		general.Wait(globals.Sem_Cpus)
		general.Wait(globals.Sem_ProcesosEnReady)

		globals.EstadosMutex.Lock()

		ordenarReadyPorRafaga()
		ejecutarUnProceso()

		globals.EstadosMutex.Unlock()
	}
}

func casoSRTdesalojo() {
	for {
		<-globals.NotificadorDesalojo // espero a que llegue una señal (bloquea hasta que llegue)

		globals.EstadosMutex.Lock()
		//log.Print("Se quiere loquear MapaProcesos en casoSRTdesalojo")
		globals.MapaProcesosMutex.Lock()
		ordenarReadyPorRafaga()
		pidEnExec, restanteExec := buscarProcesoEnExecuteDeMenorRafagaRestante()
		rafagaNuevo := globals.MapaProcesos[globals.ESTADOS.READY[0]].Rafaga.Est_Sgte
		globals.MapaProcesosMutex.Unlock()
		//log.Print("Se unloquea MapaProcesos en casoSRTdesalojo")

		if rafagaNuevo < restanteExec {
			ipCPU, puertoCPU, ok := general.BuscarCpuPorPID(pidEnExec)
			if ok {
				general.EnviarInterrupcionACPU(ipCPU, puertoCPU, pidEnExec)
				break
			} else {
				log.Printf("No se encontró la CPU que ejecuta el PID %d", pidEnExec)
			}
		}

		ejecutarUnProceso()
		globals.EstadosMutex.Unlock()
	}
}

func buscarProcesoEnExecuteDeMenorRafagaRestante() (int64, int64) {
	var pidMenorRafaga int64
	pidMenorRafaga = globals.ESTADOS.EXECUTE[0]
	for i := range globals.ESTADOS.EXECUTE {
		// Si la posicion i esta libre
		pidActual := globals.MapaProcesos[int64(i)].Pcb.Pid
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
	log.Printf("Proceso agregado a EXEC. Ahora tiene %d procesos", len(globals.ESTADOS.EXECUTE))
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

func aExecute(proceso globals.Proceso) {
	// Esto funcionaría para FIFO y SJF. Nose si SRT

	estado_anterior := proceso.Estado_Actual

	proceso = general.ActualizarMetricas(proceso, proceso.Estado_Actual)
	proceso.Estado_Actual = globals.EXECUTE
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso
	globals.ESTADOS.READY = globals.ESTADOS.READY[1:]
	globals.ESTADOS.EXECUTE = append(globals.ESTADOS.EXECUTE, proceso.Pcb.Pid)

	// LOG Cambio de Estado: ## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado %s al estado EXECUTE", proceso.Pcb.Pid, estado_anterior))
}
