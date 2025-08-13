package utils

import (
	"encoding/json"
	"fmt"
	"log/slog"

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

func InicializarMemoria() {
	tamanio := globals_memoria.MemoriaConfig.Memory_size
	tamanioPagina := globals_memoria.MemoriaConfig.Page_size
	cantidadMarcos := tamanio / tamanioPagina

	globals_memoria.Memoria = make([]byte, tamanio)
	globals_memoria.MemoriaMarcosOcupados = make([]bool, cantidadMarcos)
	globals_memoria.Procesos = make(map[int]globals_memoria.Proceso)
	globals_memoria.MetricasMap = make(map[int]globals_memoria.Memoria_Metrica)

	// Construcción de la ruta al archivo swap
	path := globals_memoria.MemoriaConfig.Swapfile_path

	// Eliminar archivo si ya existe
	if _, err := os.Stat(path); err == nil {
		err := os.Remove(path)
		if err != nil {
			slog.Debug(fmt.Sprintf("Error al eliminar swapfile existente: %v", err))
		} else {
			slog.Debug(fmt.Sprint("swapfile.bin eliminado correctamente."))
		}
	}
}
