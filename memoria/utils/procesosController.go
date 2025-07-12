package utils

import (
	"encoding/json"
	"time"

	//"fmt"
	//"bufio"
	"log"
	"net/http"

	//"os"
	//"strings"

	globals_memoria "github.com/sisoputnfrba/tp-golang/globals/memoria"
)

func IniciarProceso(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje globals_memoria.IniciarProcesoDTO
	err := decoder.Decode(&mensaje)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	log.Println("Me llego para iniciar un proceso")
	log.Printf("%+v\n", mensaje)

	// Aca empieza la logica
	var pid int = int(mensaje.Pid)
	var tamanio int = int(mensaje.Tamanio)

	var delay int64 = globals_memoria.MemoriaConfig.Memory_delay
	time.Sleep(time.Duration(delay) * time.Millisecond)
	if AlmacenarProceso(pid, tamanio, mensaje.ArchivoPseudocodigo) < 0 {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("notImplemented"))
	} else {
		log.Printf("## PID: %v - Proceso Creado - Tamaño: %v", pid, mensaje.Tamanio)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}
}

func SuspenderProceso(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje globals_memoria.PidDTO
	err := decoder.Decode(&mensaje)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	log.Printf("Me llego para suspender el proceso de pid: %d", mensaje.Pid)

	// Aca empieza la logica

	var pid int = int(mensaje.Pid)
	var delay int64 = globals_memoria.MemoriaConfig.Swap_delay
	var delayMem int64 = globals_memoria.MemoriaConfig.Memory_delay

	time.Sleep(time.Duration(delay+delayMem) * time.Millisecond)

	IncrementarMetrica("BAJADAS_SWAP", pid, 1)

	var paginas []globals_memoria.PaginaDTO

	log.Printf("Antes de llamar a eliminarMarcosFisicos")
	paginas = eliminarMarcosFisicos(pid)
	log.Print("Paginas despues de ejecutar eliminarMarcosFisicos: ", paginas)
	log.Printf("Se ejecuto eliminarMarcosFisicos")

	if escribirEnSWAP(pid, paginas) < 0 {
		log.Printf("Proceso %d no se pudo suspender por fallo al escribir en SWAP", mensaje.Pid)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Proceso no se pudo suspender por fallo al escribir en SWAP"))
		return
	}

	globals_memoria.Procesos[pid].Suspendido = true

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func FinalizarProceso(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje globals_memoria.PidDTO
	err := decoder.Decode(&mensaje)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	log.Printf("Me llego para finalizar el proceso de pid: %d", mensaje.Pid)

	// Aca empieza la logica
	var pid int = int(mensaje.Pid)

	if globals_memoria.Procesos[pid].Suspendido {
		var delay int64 = globals_memoria.MemoriaConfig.Swap_delay
		var delayMem int64 = globals_memoria.MemoriaConfig.Memory_delay

		time.Sleep(time.Duration(delay+delayMem) * time.Millisecond)
		eliminarPaginasSWAP(pid)
	} else {

		var delay int64 = globals_memoria.MemoriaConfig.Memory_delay

		time.Sleep(time.Duration(delay) * time.Millisecond)
		eliminarMarcosFisicos(pid)
	}

	delete(globals_memoria.Procesos, pid)

	var ATP int = (*globals_memoria.Metricas)[pid].AccesosTablas
	var InstSol int = (*globals_memoria.Metricas)[pid].InstruccionesSolicitadas
	var SWAP int = (*globals_memoria.Metricas)[pid].BajadasSwap
	var MemPrin int = (*globals_memoria.Metricas)[pid].SubidasMemoria
	var LecMem int = (*globals_memoria.Metricas)[pid].LecturasMemoria
	var EscMem int = (*globals_memoria.Metricas)[pid].EscriturasMemoria

	log.Printf("## PID: %d - Proceso Destruido - Métricas - Acc.T.Pag: %d; Inst.Sol.: %d; SWAP: %d; Mem.Prin.: %d; Lec.Mem.: <%d; Esc.Mem.: %d", pid, ATP, InstSol, SWAP, MemPrin, LecMem, EscMem)

	delete((*globals_memoria.Metricas), pid)
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

	log.Printf("Me llego para reanudar proceso ")
	log.Printf("%+v\n", mensaje.Pid)

	// Aca empieza la logica
	var pid int = int(mensaje.Pid)

	if globals_memoria.Procesos[pid].Suspendido == false {
		log.Printf("Proceso %d no se renaudo porque no estaba suspendido", mensaje.Pid)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("El proceso no estaba suspendido"))
		return

	}

	var delay int64 = globals_memoria.MemoriaConfig.Swap_delay
	var delayMem int64 = globals_memoria.MemoriaConfig.Memory_delay

	time.Sleep(time.Duration(delay+delayMem) * time.Millisecond)
	var paginasNecesarias int = len(globals_memoria.Procesos[pid].PaginasSWAP)

	if paginasNecesarias != 0 {
		var paginasDTO []globals_memoria.PaginaDTO
		var marcosDisponibles []int = buscarMarcosDisponibles(paginasNecesarias)
		if marcosDisponibles == nil {
			log.Printf("Proceso %d no se renaudo por falta de espacio", mensaje.Pid)
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Proceso no se renaudo por falta de espacio"))
			return
		}
		paginasDTO = eliminarPaginasSWAP(pid)
		if paginasDTO == nil {
			log.Printf("Proceso %d no se renaudo por error al eliminar paginas swap", mensaje.Pid)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("No se renaudo por error al eliminar paginas swap"))
			return
		}
		log.Print("Paginas en ReanudarProceso", paginasDTO)
		escribirPaginas(paginasDTO, marcosDisponibles)
	}
	globals_memoria.Procesos[pid].Suspendido = false
	IncrementarMetrica("SUBIDAS_MEMORIA",pid,1)

	log.Printf("Proceso %d reanudado correctamente", mensaje.Pid)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func MemoryDump(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje globals_memoria.PidDTO
	err := decoder.Decode(&mensaje)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}
	log.Printf("PID: %d - Memory Dump solicitado", mensaje.Pid)

	var delayMem int64 = globals_memoria.MemoriaConfig.Memory_delay

	time.Sleep(time.Duration(delayMem) * time.Millisecond)

	if generarMemoryDump(int(mensaje.Pid)) < 0 {
		log.Printf("Proceso %d no hizo memory dump", mensaje.Pid)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("No se hizo memory dump"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
