package utils_planifLargo

import (
	"fmt"
	"log/slog"
	"sort"
	"time"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
)

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
