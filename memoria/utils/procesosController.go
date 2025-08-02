package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"time"

	globals_memoria "github.com/sisoputnfrba/tp-golang/globals/memoria"
)

func IniciarProceso(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje globals_memoria.IniciarProcesoDTO
	err := decoder.Decode(&mensaje)
	if err != nil {
		slog.Debug(fmt.Sprintf("Error al decodificar mensaje: %s\n", err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	slog.Info(fmt.Sprint("Me llego para iniciar un proceso"))
	slog.Debug(fmt.Sprintf("%+v\n", mensaje))

	// Aca empieza la logica
	var pid int = int(mensaje.Pid)
	var tamanio int = int(mensaje.Tamanio)

	if AlmacenarProceso(pid, tamanio, mensaje.ArchivoPseudocodigo) < 0 {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("notImplemented"))
	} else {
		slog.Info(fmt.Sprintf("## PID: %d - Proceso Creado - Tamaño: %d", pid, tamanio))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}
}

func SuspenderProceso(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje globals_memoria.PidDTO
	err := decoder.Decode(&mensaje)
	if err != nil {
		slog.Debug(fmt.Sprintf("Error al decodificar mensaje: %s\n", err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	slog.Debug(fmt.Sprintf("Me llego para suspender el proceso de pid: %d", mensaje.Pid))

	// Aca empieza la logica

	var pid int = int(mensaje.Pid)
	var delay int64 = globals_memoria.MemoriaConfig.Swap_delay

	time.Sleep(time.Duration(delay) * time.Millisecond)

	proceso := globals_memoria.Procesos[pid]

	// Marco proceso suspendido
	proceso.Suspendido = true

	// Eliminas de memoria
	paginas := eliminarMarcosFisicos(pid)

	// Escribis en swap
	if escribirEnSWAP(pid, paginas) < 0 {
		slog.Debug(fmt.Sprintf("Proceso %d no se pudo suspender por fallo al escribir en SWAP", mensaje.Pid))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Proceso no se pudo suspender por fallo al escribir en SWAP"))
		return
	}

	globals_memoria.Procesos[pid] = proceso

	IncrementarMetrica("BAJADAS_SWAP", pid, 1)

	//time.Sleep(time.Duration(delay) * time.Millisecond) // PRUEBA

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func FinalizarProceso(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje globals_memoria.PidDTO
	err := decoder.Decode(&mensaje)
	if err != nil {
		slog.Debug(fmt.Sprintf("Error al decodificar mensaje: %s\n", err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	slog.Debug(fmt.Sprintf("Me llego para finalizar el proceso de pid: %d", mensaje.Pid))

	// Aca empieza la logica
	var pid int = int(mensaje.Pid)

	if globals_memoria.Procesos[pid].Suspendido {
		eliminarPaginasSWAP(pid)
	} else {

		eliminarMarcosFisicos(pid)
	}

	delete(globals_memoria.Procesos, pid)

	var ATP int = globals_memoria.MetricasMap[pid].AccesosTablas
	var InstSol int = globals_memoria.MetricasMap[pid].InstruccionesSolicitadas
	var SWAP int = globals_memoria.MetricasMap[pid].BajadasSwap
	var MemPrin int = globals_memoria.MetricasMap[pid].SubidasMemoria
	var LecMem int = globals_memoria.MetricasMap[pid].LecturasMemoria
	var EscMem int = globals_memoria.MetricasMap[pid].EscriturasMemoria

	slog.Info(fmt.Sprintf("## PID: %d - Proceso Destruido - Métricas - Acc.T.Pag: %d; Inst.Sol.: %d; SWAP: %d; Mem.Prin.: %d; Lec.Mem.: %d; Esc.Mem.: %d", pid, ATP, InstSol, SWAP, MemPrin, LecMem, EscMem))

	delete(globals_memoria.MetricasMap, pid)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func ReanudarProceso(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje globals_memoria.PidDTO
	err := decoder.Decode(&mensaje)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	//log.Printf("Me llego para reanudar proceso ")
	//log.Printf("%+v\n", mensaje.Pid)

	// Aca empieza la logica
	var pid int = int(mensaje.Pid)

	if globals_memoria.Procesos[pid].Suspendido == false {
		//log.Printf("Proceso %d no se renaudo porque no estaba suspendido", mensaje.Pid)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("El proceso no estaba suspendido"))
		return

	}

	var delay int64 = globals_memoria.MemoriaConfig.Swap_delay

	time.Sleep(time.Duration(delay) * time.Millisecond)
	cantidadPaginas := globals_memoria.Procesos[pid].CantidadDePaginas

	if cantidadPaginas != 0 {

		var marcosDisponibles []int = buscarMarcosDisponibles(cantidadPaginas)
		if marcosDisponibles == nil {
			//log.Printf("Proceso %d no se renaudo por falta de espacio", mensaje.Pid)
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Proceso no se renaudo por falta de espacio"))
			return
		}

		paginas := eliminarPaginasSWAP(pid)
		if paginas == nil {
			//log.Printf("Proceso %d no se renaudo por error al eliminar paginas swap", mensaje.Pid)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("No se renaudo por error al eliminar paginas swap"))
			return
		}
		//log.Print("Paginas en ReanudarProceso", paginas)
		paginasLinkeadas := *escribirPaginas(pid, paginas, marcosDisponibles)
		actualizarTablaPaginas(pid, paginasLinkeadas)

	}

	proceso := globals_memoria.Procesos[pid]
	proceso.Suspendido = false
	globals_memoria.Procesos[pid] = proceso

	IncrementarMetrica("SUBIDAS_MEMORIA", pid, 1)

	//log.Printf("Proceso %d reanudado correctamente", mensaje.Pid)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func MemoryDump(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje globals_memoria.PidDTO
	err := decoder.Decode(&mensaje)
	if err != nil {
		slog.Debug(fmt.Sprintf("Error al decodificar mensaje: %s\n", err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	slog.Info(fmt.Sprintf("## PID: %d - Memory Dump solicitado.", mensaje.Pid))

	var delayMem int64 = globals_memoria.MemoriaConfig.Memory_delay

	time.Sleep(time.Duration(delayMem) * time.Millisecond)

	if generarMemoryDump(int(mensaje.Pid)) < 0 {
		slog.Debug(fmt.Sprintf("Proceso %d no hizo memory dump", mensaje.Pid))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("No se hizo memory dump"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
