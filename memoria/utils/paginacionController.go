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
	var mensaje globals_memoria.SolicitudActualizarTabla
	err := decoder.Decode(&mensaje)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}
	
	actualizarTablaPaginas(int(mensaje.Pid), mensaje.Indices)
}