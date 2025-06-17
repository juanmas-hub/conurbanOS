package utils

import (
	"encoding/json"
	//"fmt"
	"log"
	"net/http"
	"os"

	globals_memoria "github.com/sisoputnfrba/tp-golang/globals/memoria"
)

func Prueba () {
	log.Printf("Prueba utilB")
}

func IniciarConfiguracion(filePath string) *globals_memoria.Memoria_Config {
	var config *globals_memoria.Memoria_Config
	configFile, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)

	return config
}

func InicializarMemoria() {
	tamanio := globals_memoria.MemoriaConfig.Memory_size
	tamanioPagina := globals_memoria.MemoriaConfig.Page_size
	cantidadMarcos := tamanio / tamanioPagina

	globals_memoria.Memoria = make([]byte, tamanio)
	globals_memoria.MemoriaMarcosOcupados = make([]bool, cantidadMarcos)

	manager := make(globals_memoria.Manager)
	globals_memoria.ProcessManager = &manager

	globals_memoria.Instrucciones = make(globals_memoria.Pseudocodigo)

	//SWAP
	globals_memoria.PaginasSwapProceso = make(map[int][]int)

	globals_memoria.ListaPaginasSwapDisponibles = make([]int, 0)

	globals_memoria.ProximaPaginaSwap = 0
}

func RecibirMensajeDeKernel(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje globals_memoria.Mensaje
	err := decoder.Decode(&mensaje)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	log.Println("Me llego un mensaje de Kernel")
	log.Printf("%+v\n", mensaje)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func RecibirMensajeDeCpu(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje globals_memoria.Mensaje
	err := decoder.Decode(&mensaje)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	log.Println("Me llego un mensaje de CPU")
	log.Printf("%+v\n", mensaje)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
