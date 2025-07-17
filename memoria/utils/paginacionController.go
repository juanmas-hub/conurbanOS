package utils

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	//"fmt"
	//"bufio"

	"net/http"

	//"os"
	//"strings"

	//globals "github.com/sisoputnfrba/tp-golang/globals/memoria"
	globals_memoria "github.com/sisoputnfrba/tp-golang/globals/memoria"
)

/*
func ActualizarTablaDePaginas(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje globals_memoria.TablaDTO
	err := decoder.Decode(&mensaje)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}
	var delayMem int64 = globals_memoria.MemoriaConfig.Memory_delay

	time.Sleep(time.Duration(delayMem) * time.Millisecond)
	actualizarTablaPaginas(int(mensaje.Pid), mensaje.Indices)

	log.Printf("Proceso %d actualizo tabla correctamente", mensaje.Pid)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}*/

/*
func ObtenerMarcoProceso(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje globals_memoria.ConsultaPaginaDTO
	err := decoder.Decode(&mensaje)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	var pid int = int(mensaje.Pid)
	var primerIndice int = int(mensaje.PrimerIndice)
	var marco int
	var delayMem int64 = globals_memoria.MemoriaConfig.Memory_delay

	time.Sleep(time.Duration(delayMem) * time.Millisecond)
	marco = obtenerMarcoDesdeTabla(pid, primerIndice)

	if marco < 0 {
		log.Printf("Error al obtener marco")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error al obtener marco"))
		return
	}

	var enviado struct {
		Dato int `json:"dato"`
	}
	enviado.Dato = marco
	jsonData, err := json.Marshal(enviado)
	if err != nil {
		log.Printf("Error al codificar la instruccion a JSON: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error interno del servidor"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)

}*/

func ObtenerMarcoProceso(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje globals_memoria.ConsultaPaginaDTO
	err := decoder.Decode(&mensaje)
	if err != nil {
		slog.Debug(fmt.Sprintf("Error al decodificar mensaje: %s\n", err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	var pid int = int(mensaje.Pid)
	var entradas []int64 = mensaje.Entradas
	var marco int
	var delayMem int64 = globals_memoria.MemoriaConfig.Memory_delay

	time.Sleep(time.Duration(delayMem) * time.Millisecond)
	marco = obtenerMarcoDesdeTabla(pid, entradas)

	if marco < 0 {
		slog.Debug(fmt.Sprintf("Error al obtener marco"))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error al obtener marco"))
		return
	}

	var enviado struct {
		Dato int `json:"marco"`
	}
	enviado.Dato = marco
	jsonData, err := json.Marshal(enviado)
	if err != nil {
		slog.Debug(fmt.Sprintf("Error al codificar la instruccion a JSON: %s", err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error interno del servidor"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)

}

func LeerPagina(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje globals_memoria.LeerPaginaDTO
	err := decoder.Decode(&mensaje)
	if err != nil {
		slog.Debug(fmt.Sprintf("Error al decodificar mensaje: %s\n", err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	var indicePagina int = int(mensaje.DireccionFisica)
	var pageSize int = int(globals_memoria.MemoriaConfig.Page_size)
	var dato string
	var delayMem int64 = globals_memoria.MemoriaConfig.Memory_delay

	time.Sleep(time.Duration(delayMem) * time.Millisecond)
	if indicePagina%pageSize != 0 {
		slog.Debug(fmt.Sprintf("Error, el indice enviado (%v) no es multiplo de %v", indicePagina, pageSize))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error indice no es multiplo del tamaño de pagina"))
		return
	}

	dato = leer(indicePagina, pageSize)

	var enviado struct {
		Dato []byte `json:"contenido"`
	}
	enviado.Dato = []byte(dato)
	jsonData, err := json.Marshal(enviado)
	if err != nil {
		slog.Debug(fmt.Sprintf("Error al codificar el mensaje a JSON: %s", err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error interno del servidor"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

func ActualizarPagina(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje globals_memoria.ActualizarPaginaDTO
	err := decoder.Decode(&mensaje)
	if err != nil {
		slog.Debug(fmt.Sprintf("Error al decodificar mensaje: %s\n", err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	slog.Debug(fmt.Sprint("Me llego para actualizar pagina: ", mensaje))

	var direccionFisica int = int(mensaje.DireccionFisica)
	var pageSize int = int(globals_memoria.MemoriaConfig.Page_size)
	var dato string = string(mensaje.Contenido)
	var delayMem int64 = globals_memoria.MemoriaConfig.Memory_delay

	time.Sleep(time.Duration(delayMem) * time.Millisecond)
	if direccionFisica%pageSize != 0 {
		slog.Debug(fmt.Sprintf("Error, el indice enviado (%v) no es multiplo de %v", direccionFisica, pageSize))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error indice no es multiplo del tamaño de pagina"))
		return
	}

	actualizarPagina(direccionFisica, dato)

	slog.Debug(fmt.Sprint("Memoria actualizada: ", globals_memoria.Memoria))

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
