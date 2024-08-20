package sysutil

import (
	"fmt"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
	"os"
	"time"
)

func GetCpuPercent() float64 {
	percent, _ := cpu.Percent(time.Second, false)
	return percent[0]
}

func GetMemPercent() float64 {
	memInfo, _ := mem.VirtualMemory()
	return memInfo.UsedPercent
}

func GetDiskPercent() float64 {
	usage, err := disk.Usage("/")
	if err != nil {
		fmt.Println("Failed to get disk usage:", err)
		return 0
	}

	percentage := float64(usage.Used) / float64(usage.Total) * 100
	return percentage
}

func GetCurrentProcess() (float64, float32) {
	proc, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		fmt.Printf("Failed to create process handle: %s\n", err)
		return 0, 0
	}
	cpuPercent, err := proc.CPUPercent()
	if err != nil {
		fmt.Printf("Failed to get CPU times: %s\n", err)
		return 0, 0
	}
	memPercent, err := proc.MemoryPercent()
	if err != nil {
		fmt.Printf("Failed to get memory info: %s\n", err)
		return 0, 0
	}
	return cpuPercent, memPercent
}
