package utils

import (
	"encoding/json"
	//"fmt"
	//"bufio"
	"log"
	"net/http"
	//"os"
	//"strings"

	//globals "github.com/sisoputnfrba/tp-golang/globals/memoria"
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

	if AlmacenarProceso(pid, mensaje.ArchivoPseudocodigo) != nil {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("notImplemented"))
	} else {
		log.Println("Proceso iniciado con exito: ", globals_memoria.Procesos[pid])

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

	var paginas []globals_memoria.PaginaDTO
	
	paginas = eliminarMarcosFisicos(pid)

	if escribirEnSWAP(pid, paginas) < 0 {
		log.Printf("Proceso %d no se pudo suspender por falo al escribir en SWAP", mensaje.Pid)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Proceso no se pudo suspender por falo al escribir en SWAP"))
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
		eliminarPaginasSWAP(pid)
	}else {
		eliminarMarcosFisicos(pid)
	}
	
	delete(globals_memoria.Procesos, pid)

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

	log.Println("Solicitud para reanudar proceso con swap")
	log.Printf("%+v\n", mensaje.Pid)

	// Aca empieza la logica
	var pid int = int(mensaje.Pid)
	var paginasNecesarias int = len(globals_memoria.Procesos[pid].PaginasSWAP)

	if paginasNecesarias != 0 {
		var paginasDTO []globals_memoria.PaginaDTO
		var marcosDisponibles []int = buscarMarcosDisponibles(paginasNecesarias)
		if marcosDisponibles == nil{
			log.Printf("Proceso %d no se renaudo por falta de espacio", mensaje.Pid)
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Proceso no se renaudo por falta de espacio"))
			return
		} 
		paginasDTO = eliminarPaginasSWAP(pid)
		if paginasDTO == nil{
			// error al eliminar las paginas SWAP
			log.Printf("Proceso %d no se renaudo por error al eliminar paginas swap", mensaje.Pid)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Proceso %d no se renaudo por error al eliminar paginas swap"))
			return
		}
		escribirPaginas(paginasDTO, marcosDisponibles)
	}
	globals_memoria.Procesos[pid].Suspendido = false

	log.Printf("Proceso %d reanudado correctamente", mensaje.Pid)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}