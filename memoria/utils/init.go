package utils

import (
	"encoding/json"
	//"fmt"
	"log"
	"os"

	globals_memoria "github.com/sisoputnfrba/tp-golang/globals/memoria"
)

func Prueba() {
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

/*
func InicializarMemoria() {
	tamanio := globals_memoria.MemoriaConfig.Memory_size
	tamanioPagina := globals_memoria.MemoriaConfig.Page_size
	cantidadMarcos := tamanio / tamanioPagina

	globals_memoria.Memoria = make([]byte, tamanio)
	globals_memoria.MemoriaMarcosOcupados = make([]bool, cantidadMarcos)

	var manager globals_memoria.Manager = make(globals_memoria.Manager)
	globals_memoria.Tablas = &manager

	globals_memoria.Procesos = make(globals_memoria.ProcesosMap)

	var metricasMap globals_memoria.MetricasMap = make(globals_memoria.MetricasMap)
	globals_memoria.Metricas = &metricasMap
	//SWAP

	globals_memoria.ListaPaginasSwapDisponibles = make([]globals_memoria.Pagina, 0)

	globals_memoria.ProximoIndiceSwap = 0
}*/

func InicializarMemoria() {
	tamanio := globals_memoria.MemoriaConfig.Memory_size
	tamanioPagina := globals_memoria.MemoriaConfig.Page_size
	cantidadMarcos := tamanio / tamanioPagina

	globals_memoria.Memoria = make([]byte, tamanio)
	globals_memoria.MemoriaMarcosOcupados = make([]bool, cantidadMarcos)

}
