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


func ActualizarTablaDePaginas(w http.ResponseWriter, r *http.Request){
	decoder := json.NewDecoder(r.Body)
	var mensaje globals_memoria.TablaDTO
	err := decoder.Decode(&mensaje)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}
	
	if actualizarTablaPaginas(int(mensaje.Pid), mensaje.Indices) != 0{
		// Error no hay suficiente espacio
		log.Printf("Proceso %d no actualizo tabla por falta de espacio", mensaje.Pid)
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Proceso no actualizo tabla por falta de espacio"))
		return
	}

	log.Printf("Proceso %d actualizo tabla correctamente", mensaje.Pid)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func ObtenerMarcoProceso(w http.ResponseWriter, r *http.Request){
	decoder := json.NewDecoder(r.Body)
	var mensaje globals_memoria.ConsultaPaginaDTO
	err := decoder.Decode(&mensaje)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	var marco int = obtenerMarcoDesdeTabla(int(mensaje.Pid), int(mensaje.PrimerIndice)) 
	
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
	
}