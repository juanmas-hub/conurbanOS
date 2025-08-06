# OS Simulation in Go

Full Operating System simulation developed in Go, structured around four main modules: **Kernel**, **Memory**, **CPU**, and **IO**. The simulation replicates fundamental behaviors of a modern OS, including process scheduling, memory management, inter-module communication, and synchronization — all implemented using Go's powerful concurrency features.

Each module is implemented as a standalone **HTTP API** using Go’s standard `net/http` package, allowing for clean inter-module communication and loose coupling. The system follows a **microservices-style architecture** where services run independently and interact over the network.

---

## 🧩 Modules

### 🧠 Kernel
The Kernel acts as the central coordinator of the system, handling process lifecycle, scheduling, and interactions between other modules.

- **Schedulers implemented**:
  - **Short-term scheduler** for CPU dispatch.
  - **Medium-term scheduler** for suspended processes.
  - **Long-term scheduler** for process admission.

### 🧮 Memory

The Memory module is responsible for managing both **virtual** and **physical memory** in the system. It handles address translation, memory allocation, and page replacement, playing a key role in the OS simulation.

- Implements a **multi-level page table** structure using **n-ary trees**, allowing efficient representation of complex virtual memory mappings.
- Supports **virtual memory** through the use of a **swap file** (a binary file on disk) to store pages that don't fit in physical memory.
- Manages **physical memory** with frame allocation, page loading, and eviction policies.
- Handles page faults and responds to memory access requests from the CPU module.
- Coordinates with the Kernel for memory initialization and process management.

### ⚙️ CPU

The CPU module simulates process execution and manages low-level memory interactions.

- Multiple CPU instances can run in **parallel**, allowing the simulation of a **multi-core environment**. Each CPU acts as an independent service, concurrently executing processes and communicating with shared modules.
- Implements a **TLB (Translation Lookaside Buffer)** for fast address translation.
- Uses **cache mechanisms** to improve memory access performance.
- Handles page faults by interacting with the Memory module.

### 🔌 IO

The IO module handles I/O-bound processes and simulates peripherals.

- Multiple IO modules can run in parallel, each simulating a separate device or set of devices. This allows concurrent handling of multiple I/O operations across the system.
- Supports basic device simulation and manages blocked processes waiting for I/O.
- Coordinates with the Kernel to reintegrate processes into the scheduling cycle after I/O completion.

---

## 🔁 Concurrency and Synchronization

The entire system takes advantage of **Go's concurrency primitives**, using:

- **Goroutines** for parallelism across modules.
- **Semaphores and mutexes** for critical sections and synchronization.

This design ensures accurate simulation of race conditions, deadlocks, and synchronization challenges in operating systems.

---

## How to run the project
    1 - Clone the Repository.
      
      git clone https://github.com/juanmas-hub/conurbanOS.git
      cd <your-repo>
    
    
    2 - Build each module and run them from separate terminals.
    memory:
    
          go build memoria
          ./memoria test_name
    
    kernel:
    
          go build kernel
          ./kernel test_name
    
    cpu:
    
          go build cpu
          ./cpu cpu_name test_name
    
    io:
    
          go build io
          ./io io_name discos
    
    io names: DISCO1, DISCO2, DISCO3, DISCO4
  
  
## Execution order

    1 - Memoria
    2 - Kernel
    3 - Cpu
    4 - Io
