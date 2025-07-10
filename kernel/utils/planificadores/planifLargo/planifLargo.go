package utils_planifLargo

import (
	"bufio"
	"fmt"
	"log"
	"log/slog"
	"os"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
	estados "github.com/sisoputnfrba/tp-golang/kernel/utils/estados"
	general "github.com/sisoputnfrba/tp-golang/kernel/utils/general"
	cp "github.com/sisoputnfrba/tp-golang/kernel/utils/planificadores/planifCorto"
)

func IniciarPlanificadorLargoPlazo(archivo string, tamanio int64) {
	// Espera el Enter en otra rutina asi se puede abrir el servidor
	reader := bufio.NewReader(os.Stdin)
	slog.Info(" ---- Planificador de largo plazo en STOP, presionar ENTER: ")
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
	go pasarProcesosAReady()

	CrearProcesoNuevo(archivo, tamanio) // Primer proceso
}

func pasarProcesosAReady() {
	// Ejecuta en un hilo
	// Esta funcion se ejecuta cuando llega un proceso a NEW, a EXIT, a SUSP_BLOCKED y a SUSP_READY
	// Voy a intentar pasar la mayor cantidad de procesos que pueda mientras memoria tenga espacio

	for {
		//general.Wait(globals.Sem_PasarProcesoAReady)
		globals.WaitPasarProcesoAReady()
		if globals.PLANIFICADOR_LARGO_PLAZO_BLOCKED == false {
			slog.Debug(fmt.Sprintf("Intentando pasar procesos a ready porque llego un proceso a:  %s", globals.DeDondeSeLlamaPasarProcesosAReady))

			var lenghtSUSP_READY = len(globals.ESTADOS.SUSP_READY)
			for lenghtSUSP_READY > 0 {
				pid := globals.ESTADOS.SUSP_READY[0]
				if general.SolicitarInicializarProcesoAMemoria_DesdeSUSP_READY(pid) == false {
					break
				}

				globals.MapaProcesosMutex.Lock()
				proceso := globals.MapaProcesos[pid]
				globals.MapaProcesosMutex.Unlock()
				estados.SuspReadyAReady(proceso)
				lenghtSUSP_READY--
			}

			if lenghtSUSP_READY == 0 {

				for len(globals.ESTADOS.NEW) > 0 {
					globals.EstadosMutex.Lock()
					procesoNuevo := globals.ESTADOS.NEW[0]
					globals.EstadosMutex.Unlock()
					//slog.Debug(fmt.Sprintf("Solicito iniciar proceso: %d", procesoNuevo.Proceso.Pcb.Pid))
					if general.SolicitarInicializarProcesoAMemoria_DesdeNEW(procesoNuevo) == false {
						break
					}

					globals.EstadosMutex.Lock()
					globals.ESTADOS.NEW = globals.ESTADOS.NEW[1:]
					globals.EstadosMutex.Unlock()
					go estados.NewAReady(procesoNuevo)

				}
			}
		}
	}
}
