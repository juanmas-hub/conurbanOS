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

func ConsultarMock(w http.ResponseWriter, r *http.Request) {
	mock := CalcularMock()

	var enviado struct {
		Mock int `json:"mock"`
	}

	enviado.Mock = mock

	jsonData, err := json.Marshal(enviado)

	if (err != nil){
		log.Printf("Error al codificar el mock a JSON: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error interno del servidor"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)

}

func EnviarInstruccion(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje globals_memoria.InstruccionDTO
	err := decoder.Decode(&mensaje)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	log.Printf("Solicitud de instruccion de PID: %d y PC: %d", mensaje.Pid, mensaje.Pc)
	log.Printf("%+v\n", mensaje.Pid)

	instruccion := globals_memoria.Procesos[int(mensaje.Pid)].Pseudocodigo[mensaje.Pc]
	var enviado struct {
		Instruccion string `json:"instruccion"`
	}
	enviado.Instruccion = instruccion
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

func AccederEspacioUsuarioLectura(w http.ResponseWriter, r *http.Request){
	decoder := json.NewDecoder(r.Body)
	var mensaje globals_memoria.LecturaDTO
	err := decoder.Decode(&mensaje)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	var leido string = leer(int(mensaje.Posicion), int(mensaje.Tamanio))

	var enviado struct {
		Dato string `json:"dato"`
	}
	enviado.Dato = leido
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

func AccederEspacioUsuarioEscritura(w http.ResponseWriter, r *http.Request){
	decoder := json.NewDecoder(r.Body)
	var mensaje globals_memoria.EscrituraDTO
	err := decoder.Decode(&mensaje)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	if escribir(int(mensaje.Posicion), mensaje.Dato) < 0{
		log.Printf("Error al escribir en la posicion %v", int(mensaje.Posicion))
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Error al escribir en la posicion"))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
