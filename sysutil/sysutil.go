package sysutil

import (
	"fmt"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/zp857/goutil/constants"
	"log"
	"os"
	"runtime"
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

func GetDiskIO() (uint64, uint64) {
	if runtime.GOOS == constants.Windows {
		return 0, 0
	}
	// 获取磁盘 I/O 信息
	ioStat, err := disk.IOCounters()
	if err != nil {
		return 0, 0
	}
	time.Sleep(3 * time.Second)
	ioStats2, err := disk.IOCounters()
	if err != nil {
		return 0, 0
	}
	var readBytes uint64
	var writeBytes uint64
	// 计算 I/O 占用率
	for _, io1 := range ioStat {
		for _, io2 := range ioStats2 {
			if io2.Name == io1.Name {
				if io2.ReadBytes != 0 && io1.ReadBytes != 0 && io2.ReadBytes > io1.ReadBytes {
					readBytes += uint64(float64(io2.ReadBytes-io1.ReadBytes) / 60)
				}
				if io2.WriteBytes != 0 && io1.WriteBytes != 0 && io2.WriteBytes > io1.WriteBytes {
					writeBytes += uint64(float64(io2.WriteBytes-io1.WriteBytes) / 60)
				}
				break
			}
		}
	}
	return readBytes, writeBytes
}

func GetNetIO() float64 {
	// 获取网络 I/O 信息
	netStats1, err := net.IOCounters(false)
	if err != nil {
		log.Fatal(err)
	}
	time.Sleep(1 * time.Second)
	netStats2, err := net.IOCounters(false)
	if err != nil {
		log.Fatal(err)
	}

	// 计算带宽占用
	for _, stat1 := range netStats1 {
		for _, stat2 := range netStats2 {
			if stat1.Name == stat2.Name {
				bytesSent := stat2.BytesSent - stat1.BytesSent
				bytesRecv := stat2.BytesRecv - stat1.BytesRecv
				fmt.Printf("Interface: %s, Bytes Sent: %d, Bytes Received: %d\n", stat1.Name, bytesSent, bytesRecv)
			}
		}
	}
	return 0
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
