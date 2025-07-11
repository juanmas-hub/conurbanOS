package planificadores

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"time"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
	general "github.com/sisoputnfrba/tp-golang/kernel/utils/general"
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

	go EjecutarPlanificadorCortoPlazo()

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
				SuspReadyAReady(proceso)
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
					go NewAReady(procesoNuevo)

				}
			}
		}
	}
}

func ordenarNewPorTamanio() {

	// Con ordenar por tamaño (mas chicho primero) ya el algoritmo PMCP estaria hecho (creo)
	sort.Slice(globals.ESTADOS.NEW, func(i, j int) bool {
		return globals.ESTADOS.NEW[i].Tamaño < globals.ESTADOS.NEW[j].Tamaño
	})
}

func CrearProcesoNuevo(archivo string, tamanio int64) {

	globals.PIDCounterMutex.Lock()

	pid := globals.PIDCounter
	globals.PIDCounter++

	globals.PIDCounterMutex.Unlock()

	proceso := globals.Proceso{
		Pcb: globals.PCB{
			Pid: pid,
			PC:  0,
			// Las metricas las inicializa en 0
		},
		Estado_Actual:        globals.NEW,
		Rafaga:               nil,
		UltimoCambioDeEstado: time.Now(),
	}

	if globals.KernelConfig.Scheduler_algorithm != "FIFO" {
		rafaga := globals.Rafagas{
			Raf_Ant:  0,
			Est_Ant:  0,
			Est_Sgte: float64(globals.KernelConfig.Initial_estimate),
		}

		proceso.Rafaga = &rafaga
	}

	globals.MapaProcesosMutex.Lock()
	globals.MapaProcesos[pid] = proceso
	globals.MapaProcesosMutex.Unlock()

	procesoNuevo := globals.Proceso_Nuevo{
		Archivo_Pseudocodigo: archivo,
		Tamaño:               tamanio,
		Proceso:              proceso,
	}

	globals.EstadosMutex.Lock()

	// Aca no hay metricas que actualizar
	globals.ESTADOS.NEW = append(globals.ESTADOS.NEW, procesoNuevo)

	// LOG Creación de Proceso: “## (<PID>) Se crea el proceso - Estado: NEW”
	slog.Info(fmt.Sprintf("## (%d) Se crea el proceso - Estado: NEW", pid))

	//log.Printf("Después de agregar, NEW tiene %d procesos", len(globals.ESTADOS.NEW))
	if globals.KernelConfig.New_algorithm == "PMCP" {
		ordenarNewPorTamanio()
		//log.Print("NEW despues de ordenarlo: ", globals.ESTADOS.NEW)
	}

	globals.EstadosMutex.Unlock()

	globals.DeDondeSeLlamaMutex.Lock()
	globals.DeDondeSeLlamaPasarProcesosAReady = "New"
	globals.DeDondeSeLlamaMutex.Unlock()
	globals.SignalPasarProcesoAReady()

}

func FinalizarProceso(pid int64) {
	globals.MapaProcesosMutex.Lock()
	proceso, ok := globals.MapaProcesos[pid]
	globals.MapaProcesosMutex.Unlock()
	if !ok {
		log.Printf("No se encontró el proceso con PID %d", pid)
		return
	}

	// Enviar a memoria
	ok = enviarFinalizacionDeProceso_AMemoria(globals.KernelConfig.Ip_memory, globals.KernelConfig.Port_memory, pid)
	if !ok {
		log.Printf("Memoria no pudo finalizar el proceso PID %d.", pid)
		return
	}

	// Mover a EXIT y eliminar de su cola
	ProcesoAExit(proceso)

	globals.EstadosMutex.Lock()
	EliminarProcesoDeSuCola(pid, proceso.Estado_Actual)
	globals.EstadosMutex.Unlock()

	// Eliminar del mapa de procesos
	globals.MapaProcesosMutex.Lock()
	delete(globals.MapaProcesos, pid)
	globals.MapaProcesosMutex.Unlock()

	// Señal para ready
	globals.DeDondeSeLlamaMutex.Lock()
	globals.DeDondeSeLlamaPasarProcesosAReady = "Exit"
	globals.DeDondeSeLlamaMutex.Unlock()
	globals.SignalPasarProcesoAReady()
}

func enviarFinalizacionDeProceso_AMemoria(ip string, puerto int64, pid int64) bool {
	mensaje := globals.PidJSON{PID: pid}
	body, err := json.Marshal(mensaje)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	// Posible problema con el int64 del puerto
	url := fmt.Sprintf("http://%s:%d/finalizarProceso", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", ip, puerto)
	}

	slog.Debug(fmt.Sprintf("Finalizacion PID %d enviada a memoria, respuesta: %s", pid, resp.Status))

	if resp.StatusCode == http.StatusOK {
		return true
	}
	return false
}
