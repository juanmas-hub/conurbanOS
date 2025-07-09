package utils_planifLargo

import (
	"bufio"
	"log"
	"os"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
	estados "github.com/sisoputnfrba/tp-golang/kernel/utils/estados"
	general "github.com/sisoputnfrba/tp-golang/kernel/utils/general"
	cp "github.com/sisoputnfrba/tp-golang/kernel/utils/planificadores/planifCorto"
)

func IniciarPlanificadorLargoPlazo(archivo string, tamanio int64) {
	// Espera el Enter en otra rutina asi se puede abrir el servidor
	reader := bufio.NewReader(os.Stdin)
	log.Println("Planificador de largo plazo en STOP, presionar ENTER: ")
	for {
		text, _ := reader.ReadString('\n')
		log.Print(text)

		if text == "\n" {
			globals.PLANIFICADOR_LARGO_PLAZO_BLOCKED = false
			break
		}
	}

	go cp.EjecutarPlanificadorCortoPlazo()

	// El planif largo tiene dos partes:
	go escucharFinalizacionesDeProcesos()
	go pasarProcesosAReady()

	CrearProcesoNuevo(archivo, tamanio) // Primer proceso
}

func pasarProcesosAReady() {
	// Ejecuta en un hilo
	// Esta funcion se ejecuta cuando llega un proceso a NEW, a EXIT, a SUSP_BLOCKED y a SUSP_READY
	// Voy a intentar pasar la mayor cantidad de procesos que pueda mientras memoria tenga espacio

	for {
		general.Wait(globals.Sem_PasarProcesoAReady)

		log.Print("Intentando pasar procesos a ready porque llego un proceso a: ", globals.DeDondeSeLlamaPasarProcesosAReady)

		var lenghtSUSP_READY = len(globals.ESTADOS.SUSP_READY)
		for lenghtSUSP_READY > 0 {
			pid := globals.ESTADOS.SUSP_READY[0]
			if general.SolicitarInicializarProcesoAMemoria_DesdeSUSP_READY(pid) == false {
				break
			}

			//log.Print("Se quiere loquear MapaProcesos en PasarProcesosAReady x1")
			globals.MapaProcesosMutex.Lock()
			//log.Print("Se loquea MapaProcesos en PasarProcesosAReady x1")
			proceso := globals.MapaProcesos[pid]
			globals.MapaProcesosMutex.Unlock()
			//log.Print("Se unloquea MapaProcesos en PasarProcesosAReady x1")
			estados.SuspReadyAReady(proceso)
			lenghtSUSP_READY--
		}

		if lenghtSUSP_READY == 0 {

			for len(globals.ESTADOS.NEW) > 0 {
				//log.Print("Se quiere bloquear en pasarProcesosAReady")
				globals.EstadosMutex.Lock()
				//log.Print("Se bloqueo en pasarProcesosAReady")
				procesoNuevo := globals.ESTADOS.NEW[0]
				//log.Print("Se quiere desbloquear en pasarProcesosAReady")
				globals.EstadosMutex.Unlock()
				//log.Print("Se desbloqueo en pasarProcesosAReady")
				if general.SolicitarInicializarProcesoAMemoria_DesdeNEW(procesoNuevo) == false {
					break
				}

				estados.NewAReady(procesoNuevo)

			}
		}
	}
}

func escucharFinalizacionesDeProcesos() {
	// Queda escuchando en un hilo los procesos que terminan
	for {
		general.Wait(globals.Sem_ProcesoAFinalizar)
		globals.ProcesosAFinalizarMutex.Lock()
		pid := globals.ProcesosAFinalizar[0]
		globals.ProcesosAFinalizar = globals.ProcesosAFinalizar[1:]
		globals.ProcesosAFinalizarMutex.Unlock()
		go finalizarProceso(pid)
	}
}
