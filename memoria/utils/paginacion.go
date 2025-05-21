package utils

import (
	//"encoding/json"
	"fmt"
	"log"
	//"net/http"
	//"os"

	globals_memoria "github.com/sisoputnfrba/tp-golang/globals/memoria"
)


func crearTabla(entradasPorPagina int64) *globals_memoria.TablaDePaginas {
	return &globals_memoria.TablaDePaginas{
		Entradas: make([]globals_memoria.EntradaTablaPagina, entradasPorPagina),
	}
}

func verificarPIDUnico(pid int) error {
	if _, existe := (*globals_memoria.ProcessManager)[pid]; existe {
		return fmt.Errorf("el proceso con PID %d ya existe", pid)
	}
	return nil
}

func extraerIndices(pagina int) []int {

	// Para una estructura de 5 niveles con 4 entradas por página,
	// necesitamos extraer el índice para cada nivel
	// Con 4 entradas por nivel, cada índice ocupa 2 bits (valores de 0-3)
	// Se toman 10 bytes finales de los 32 bytes del numero int

	// Extraer los índices para cada nivel (de más significativo a menos significativo)
	indiceNivel1 := (pagina >> 8) & 0x3 // bits 9-8
	indiceNivel2 := (pagina >> 6) & 0x3 // bits 7-6
	indiceNivel3 := (pagina >> 4) & 0x3 // bits 5-4
	indiceNivel4 := (pagina >> 2) & 0x3 // bits 3-2
	indiceNivel5 := pagina & 0x3        // bits 1-0

	log.Printf("IndiceNivel1: %d", indiceNivel1)
	log.Printf("IndiceNivel2: %d", indiceNivel2)
	log.Printf("IndiceNivel3: %d", indiceNivel3)
	log.Printf("IndiceNivel4: %d", indiceNivel4)
	log.Printf("IndiceNivel5: %d", indiceNivel5)

	var indicesNiveles []int = []int{indiceNivel1, indiceNivel2, indiceNivel3, indiceNivel4, indiceNivel5}
	return indicesNiveles
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

func actualizarTablaPaginas(pid int, pagina int, marco int) {

	var ENTRIES_PER_PAGE int64 = globals_memoria.MemoriaConfig.Entries_per_page
	var NUMBER_OF_LEVELS int = int(globals_memoria.MemoriaConfig.Number_of_levels)

	var tablaActual *globals_memoria.TablaDePaginas = (*globals_memoria.ProcessManager)[pid]
	var indicesNiveles []int = append(extraerIndices(pagina), marco)

	for i := 0; i < NUMBER_OF_LEVELS; i++ {
		var indiceActual int = indicesNiveles[i]
		var indiceSiguiente int = indicesNiveles[i+1]

		if tablaActual.Entradas[indiceActual].SiguienteNivel == nil {
			tablaActual.Entradas[indiceActual].SiguienteNivel = crearTabla(ENTRIES_PER_PAGE)
		}

		tablaActual.Entradas[indiceActual].Pagina = indiceActual
		tablaActual.Entradas[indiceActual].Marco = indiceSiguiente

		log.Printf("Nivel %d: pagina %d y marco %d", i+1, tablaActual.Entradas[indiceActual].Pagina, tablaActual.Entradas[indiceActual].Marco)

		if i != NUMBER_OF_LEVELS-1 {
			tablaActual = tablaActual.Entradas[indiceActual].SiguienteNivel
		}
	}

	fmt.Printf("Página %d mapeada al marco %d. Ruta de índices: %d->%d->%d->%d->%d\n",
		pagina, marco, indicesNiveles[0], indicesNiveles[1], indicesNiveles[2], indicesNiveles[3], indicesNiveles[4])

}

func AlmacenarProceso(pid int, instrucciones []string) int {

	verificarPIDUnico(pid)

	var PAGE_SIZE int = int(globals_memoria.MemoriaConfig.Page_size)
	var ENTRIES_PER_PAGE int64 = globals_memoria.MemoriaConfig.Entries_per_page

	var paginasNecesarias int = 0

	for _, instruccion := range instrucciones {
		// Redondeado hacia arriba
		paginasNecesarias += ((len(instruccion) + PAGE_SIZE - 1) / PAGE_SIZE)
	}

	var marcosDisponibles []int = buscarMarcosDisponibles(paginasNecesarias)

	if marcosDisponibles == nil {
		// swap()
		log.Println("Proceso guardado en swap")
		return 0
	}

	log.Println("Crear la tabla de páginas para el proceso")
	tablaPrincipal := crearTabla(ENTRIES_PER_PAGE)

	(*globals_memoria.ProcessManager)[pid] = tablaPrincipal

	// Almacenar instrucciones en memoria física
	var offset int = 0
	var paginaActual int = 0
	var marcoFisico int = marcosDisponibles[paginaActual]

	actualizarTablaPaginas(pid, paginaActual, marcoFisico)

	for _, instruccion := range instrucciones {
		var bytesInstruccion []byte = []byte(instruccion)

		for i := 0; i < len(bytesInstruccion); i++ {

			// Si llenamos una página, pasamos a la siguiente
			if offset >= PAGE_SIZE {
				paginaActual++
				offset = 0

				marcoFisico = marcosDisponibles[paginaActual]
				actualizarTablaPaginas(pid, paginaActual, marcoFisico)
			}

			posicionFisica := (marcoFisico * PAGE_SIZE) + offset

			globals_memoria.Memoria[posicionFisica] = bytesInstruccion[i]

			globals_memoria.MemoriaMarcosOcupados[marcoFisico] = true

			offset++
		}
	}
	log.Println("Proceso almacenado en memoria fisica")

	fmt.Printf("Proceso %d almacenado correctamente utilizando %d páginas\n", pid, paginasNecesarias)
	return 0
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

