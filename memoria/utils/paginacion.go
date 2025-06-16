package utils

import (
	//"encoding/json"
	"fmt"
	"log"
	//"net/http"
	//"os"

	globals_memoria "github.com/sisoputnfrba/tp-golang/globals/memoria"
)

func leer(direccion int, tamanio int) string {
	var leido string = ""

	for i := 0; i < tamanio; i++ {
		leido += string(globals_memoria.Memoria[direccion + i])
	}
	return leido
}

func escribir(direccion int, dato string){

	// var marco int = direccion / int(globals_memoria.MemoriaConfig.Page_size) // redondea hacia abajo

	for i := 0; i < len(dato); i++{
		globals_memoria.Memoria[direccion + i] = dato[i]
	}
}

func crearTabla(entradasPorPagina int64) *globals_memoria.TablaDePaginas {
	return &globals_memoria.TablaDePaginas{
		Entradas: make([]globals_memoria.EntradaTablaPagina, entradasPorPagina),
	}
}

func verificarPIDUnico(pid int) int {
	if _, existe := (*globals_memoria.ProcessManager)[pid]; existe {
		return 1
	}
	return 0
}



func buscarMarcosDisponibles(cantidad int) []int {
	var result []int = make([]int, 0, cantidad)

	for i := 0; i < len(globals_memoria.MemoriaMarcosOcupados); i++ {
		if !globals_memoria.MemoriaMarcosOcupados[i] {
			result = append(result, i)
			if len(result) >= cantidad {
				return result
			}
		}
	}

	log.Printf("no hay suficientes marcos libres: se encontraron %d de %d", len(result), cantidad)
	return nil
}

func actualizarTablaPaginas(pid int, indices []int) {
	var ENTRIES_PER_PAGE int64 = globals_memoria.MemoriaConfig.Entries_per_page
	// var NUMBER_OF_LEVELS int = int(globals_memoria.MemoriaConfig.Number_of_levels)

	var tablaActual *globals_memoria.TablaDePaginas = (*globals_memoria.ProcessManager)[pid]

	for i := 0; i < len(indices)-1; i++ {
		var indiceActual int = indices[i]
		var indiceSiguiente int = indices[i + 1]

		if tablaActual.Entradas[indiceActual].SiguienteNivel == nil {
			tablaActual.Entradas[indiceActual].SiguienteNivel = crearTabla(ENTRIES_PER_PAGE)
		}

		tablaActual.Entradas[indiceActual].Pagina = indiceActual
		tablaActual.Entradas[indiceActual].Marco = indiceSiguiente
		

		log.Printf("Nivel %d: pagina %d y marco %d", i+1, tablaActual.Entradas[indiceActual].Pagina, tablaActual.Entradas[indiceActual].Marco)

		tablaActual = tablaActual.Entradas[indiceActual].SiguienteNivel
		
	}

	fmt.Printf("Ruta de índices: %d->%d->%d->%d->%d\n", indices[0], indices[1], indices[2], indices[3], indices[4])

}

func AlmacenarProceso(pid int, filename string) error {

	if verificarPIDUnico(pid) != 0 {
		fmt.Errorf("el proceso con PID %d ya existe", pid)
	}

	var ENTRIES_PER_PAGE int64 = globals_memoria.MemoriaConfig.Entries_per_page
	var instrucciones []string

	instrucciones = ObtenerInstruccionesDesdeArchivo(filename)

	globals_memoria.Instrucciones[pid] = instrucciones
	
	(*globals_memoria.ProcessManager)[pid] = crearTabla(ENTRIES_PER_PAGE)


	log.Println("Proceso almacenado")

	return nil
}

func obtenerMarcoDesdeTabla(pid int, pagina int) int {
	var NUMBER_OF_LEVELS int = int(globals_memoria.MemoriaConfig.Number_of_levels)

	var indicesNiveles []int = extraerIndices(pagina)

	var tablaActual *globals_memoria.TablaDePaginas = (*globals_memoria.ProcessManager)[pid]

	for i := 0; i < NUMBER_OF_LEVELS-1; i++ {
		indiceActual := indicesNiveles[i]
		tablaActual = tablaActual.Entradas[indiceActual].SiguienteNivel
	}

	// En el último nivel, obtener el marco físico
	ultimoIndice := indicesNiveles[NUMBER_OF_LEVELS-1]

	return tablaActual.Entradas[ultimoIndice].Marco

}

