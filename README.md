# OS Simulation in Go

This repository contains a full Operating System simulation developed in Go, structured around four main modules: **Kernel**, **Memory**, **CPU**, and **IO**. The simulation replicates fundamental behaviors of a modern OS, including process scheduling, memory management, inter-module communication, and synchronization — all implemented using Go's powerful concurrency features.

---

## 🧩 Modules Overview

### 🧠 Kernel
The Kernel acts as the central coordinator of the system, handling process lifecycle, scheduling, and interactions between other modules.

- **Schedulers implemented**:
  - **Short-term scheduler** for CPU dispatch.
  - **Medium-term scheduler** for suspended processes.
  - **Long-term scheduler** for process admission.
- Communicates with Memory and CPU via sockets or channels.
- Implements semaphores and mutexes for inter-process synchronization.

### 🧮 Memory
The Memory module is responsible for managing process address spaces and page tables.

- **Page tables** implemented as **n-ary trees** to efficiently represent hierarchical memory structures.
- Supports dynamic memory allocation, frame management, and page replacement policies.
- Interfaces with CPU and Kernel for memory access and swapping.

### ⚙️ CPU
The CPU module simulates process execution and manages low-level memory interactions.

- Implements **TLB (Translation Lookaside Buffer)** for fast address translation.
- Uses **cache mechanisms** to improve memory access performance.
- Handles page faults by interacting with Memory.
- Uses **Go routines** and **channels** for concurrent execution.

### 🔌 IO
The IO module handles I/O-bound processes and simulates peripherals.

- Supports basic device simulation and manages blocked processes.
- Coordinates with the Kernel to reintegrate processes after I/O completion.

---

## 🔁 Concurrency and Synchronization

The entire system takes advantage of **Go's concurrency primitives**, using:

- **Goroutines** for parallelism across modules.
- **Channels** for communication between components.
- **Semaphores and mutexes** for critical sections and synchronization.

This design ensures accurate simulation of race conditions, deadlocks, and synchronization challenges in operating systems.

---

## 🚀 Technologies

- **Language**: Go (Golang)
- **Concurrency**: Goroutines, channels, semaphores
- **Architecture**: Modular (Kernel, Memory, CPU, IO)
- **Data structures**: N-ary trees, queues, buffers

## Execution order

    1 - Memoria
    2 - Kernel
    3 - Cpu
    4 - Io
