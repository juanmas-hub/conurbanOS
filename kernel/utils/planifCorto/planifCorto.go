package utils_planifCorto

import (
	"log"
	"sort"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
	general "github.com/sisoputnfrba/tp-golang/kernel/utils/general"
)

// ----- FUNCIONES EXPORTADAS -------

func EjecutarPlanificadorCortoPlazo() {
	// Es un while infinito, pero se queda esperando al principio a que haya CPUs libres (wait es bloqueante)
	// Cuando reciba que hay CPUs disponibles, va a Ready y se fija si hay procesos para pasar a execute
	// No se que hacer cuando no hay procesos en Ready

	if globals.KernelConfig.Scheduler_algorithm == "FIFO" {
		for {
			general.Wait(globals.Sem_Cpus)            // Espero a que haya Cpus libres
			general.Wait(globals.Sem_ProcesosEnReady) // Espero a que haya procesos en Ready
			globals.EstadosMutex.Lock()

			// Esto lo hago asi para probarlo,
			if len(globals.ESTADOS.READY) > 0 {

				procesoAEjecutar := globals.ESTADOS.READY[0]

				ip, port := elegirCPUlibre()

				enviarProcesoAEjecutar_ACPU(ip, port, procesoAEjecutar)

				globals.MapaProcesosMutex.Lock()

				readyAExecute(globals.MapaProcesos[procesoAEjecutar])
				log.Printf("Proceso agregado a EXEC. Ahora tiene %d procesos", len(globals.ESTADOS.EXECUTE))

				globals.EstadosMutex.Unlock()
				globals.MapaProcesosMutex.Unlock()
			}
		}
	}

	// Modifique solamente el de FIFO, hay que modificar los de aca abajo (semaforos)

	if globals.KernelConfig.Scheduler_algorithm == "SJF" {
		for {
			general.Wait(globals.Sem_Cpus)
			general.Wait(globals.Sem_ProcesosEnReady)

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
			ip, port := elegirCPUlibre()
			enviarProcesoAEjecutar_ACPU(ip, port, procesoAEjecutar)
			globals.MapaProcesosMutex.Lock()
			readyAExecute(globals.MapaProcesos[procesoAEjecutar])
			log.Printf("Proceso agregado a EXEC. Ahora tiene %d procesos", len(globals.ESTADOS.EXECUTE))
			globals.EstadosMutex.Unlock()
			globals.MapaProcesosMutex.Unlock()
		}
	}

	if globals.KernelConfig.Scheduler_algorithm == "SRT" {
		for {
			general.Wait(globals.Sem_Cpus)
			general.Wait(globals.Sem_ProcesosEnReady)

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
					// Hay
					// esto seria una funcion
					//for _, cpu := range globals.ListaCPUs {
					//	if cpu.EstaLibre {
					//		return false
					//	}
					//}
					//return true
					cpu := globals.ListaCPUs[0].Handshake
					ipCPU := cpu.IP
					puertoCPU := cpu.Puerto
					general.EnviarInterrupcionACPU(ipCPU, puertoCPU, pidEnExec)
					// Aca la logica para mandar el proceso con rafaga mas corta - despues lo hago me voy a tocar (la guitarra)
				}
			}
			// Si no hay ningun proceso en EXECUTE -> simplemente agregamos el primero de READY
			procesoAEjecutar := globals.ESTADOS.READY[0]
			ip, port := elegirCPUlibre()
			enviarProcesoAEjecutar_ACPU(ip, port, procesoAEjecutar)
			globals.MapaProcesosMutex.Lock()
			readyAExecute(globals.MapaProcesos[procesoAEjecutar])
			log.Printf("Proceso agregado a EXEC. Ahora tiene %d procesos", len(globals.ESTADOS.EXECUTE))
			globals.EstadosMutex.Unlock()
			globals.MapaProcesosMutex.Unlock()
		}
	}
}

// ----- FUNCIONES LOCALES -------

func actualizarEstimado(pid int64, rafagaReal int64) {
	// Me imagino que esto se usa cuando se termina de ejecutar un proceso

	proceso := globals.MapaProcesos[pid]
	alpha := globals.KernelConfig.Alpha
	ant := proceso.Rafaga.Est_Sgte
	proceso.Rafaga.Est_Sgte = rafagaReal*alpha + ant*(1-alpha)
	// Est(n+1) =  R(n) + (1-) Est(n) ;    [0,1]

	globals.MapaProcesos[pid] = proceso
}

func elegirCPUlibre() (string, int64) {
	// Hay que hacerlo. Seguramente haya que cambiar HandshakesCPU para indicar cual esta libre
	// Recorremos la lista
	for i := range globals.ListaCPUs {
		// Si la posicion i esta libre
		if globals.ListaCPUs[i].EstaLibre {
			// La marcamos como ocupada
			globals.ListaCPUs[i].EstaLibre = false
			// Devolvemos IP y PUERTO
			return globals.ListaCPUs[i].Handshake.IP, globals.ListaCPUs[i].Handshake.Puerto
		}
	}
	// Si devuelve esto hay un error, porque esta funcion se tiene que ejecutar cuando el semaforo lo permita
	log.Println("No se encontro CPU libre")
	return "", -1
}

func readyAExecute(proceso globals.Proceso) {
	// Esto funcionaría para FIFO y SJF. Nose si SRT

	proceso.Estado_Actual = globals.EXECUTE
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso
	globals.ESTADOS.READY = globals.ESTADOS.READY[1:]
	globals.ESTADOS.EXECUTE = append(globals.ESTADOS.EXECUTE, proceso.Pcb.Pid)
}

func enviarProcesoAEjecutar_ACPU(ip string, puerto int64, pid int64) {
	/*mensaje := globals.PidJSON{PID: pid}
	body, err := json.Marshal(mensaje)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	// Posible problema con el int64 del puerto
	url := fmt.Sprintf("http://%s:%d/dispatchProceso", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", ip, puerto)
	}

	log.Printf("respuesta del servidor: %s", resp.Status)*/
}
