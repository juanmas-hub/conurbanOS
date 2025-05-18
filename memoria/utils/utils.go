package utils

import (
	"encoding/json"
	//"fmt"
	"log"
	"net/http"
	//"os"

	globals_memoria "github.com/sisoputnfrba/tp-golang/globals/memoria"
)

func CalcularMock() int {
	PAGE_SIZE := int(globals_memoria.MemoriaConfig.Page_size)
	libres := 0

	for _, estaOcupado := range globals_memoria.MemoriaMarcosOcupados {
		if !estaOcupado {
			libres++
		}
	}
	return libres * PAGE_SIZE
}

func ConsultarMock(w http.ResponseWriter, r *http.Request) {
	mock := CalcularMock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(map[string]int{
		"espacio_libre": mock,
	})
}

func IniciarProceso(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje globals_memoria.SolicitudIniciarProceso
	err := decoder.Decode(&mensaje)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	log.Println("Me llego para iniciar un proceso")
	log.Printf("%+v\n", mensaje)

	// Aca tenes que hacer lo que sea para iniciar
	// Si pudiste iniciar el proceso => devolve http.StatusOK
	// Sino devolve error

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
